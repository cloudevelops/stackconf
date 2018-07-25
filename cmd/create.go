// Copyright © 2017 Zdenek Janda <zdenek.janda@cloudevelops.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"database/sql"
	"encoding/json"
	"html"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	_ "github.com/go-sql-driver/mysql"
	//"github.com/davecgh/go-spew/spew"

	"github.com/cloudevelops/go-foreman"
	jenkins "github.com/cloudevelops/go-jenkins"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	//"bytes"
	"bufio"
	"fmt"

	"github.com/cloudevelops/go-powerdns"
)

var p *powerdns.Powerdns
var d *sql.DB
var hostFqdn string
var hostName string
var domainName string
var ipAddress string
var j *jenkins.Jenkins
var puppetSslError bool
var f *foreman.Foreman

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new stackconf host",
	Long:  `Create a new stackconf host.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debugf("Create command: starting")
		//Foreman prototype
		f = foreman.NewForeman(viper.GetString("foreman.config.host"), viper.GetString("foreman.config.username"), viper.GetString("foreman.config.password"))
		// Host
		puppetVersion := viper.GetInt("puppet.version")
		spew.Dump(puppetVersion)
		if puppetVersion == 4 {
			hostFqdn = viper.GetString("puppetfacter.networking.fqdn")
		} else {
			hostFqdn = viper.GetString("puppetfacter.fqdn")
		}
		hostNameSplit := strings.Split(hostFqdn, ".")
		hostName = hostNameSplit[0]
		domainName = strings.Replace(hostFqdn, hostName+".", "", -1)
		err := foremanDelete(hostFqdn)
		if err != nil {
			log.Debugf("Foreman failed to delete host !")
		}
		// Domain
		domain, err := f.SearchResource("domains", domainName)
		var domainId string
		if err == nil {
			domainId = strconv.FormatFloat(domain["id"].(float64), 'f', -1, 64)
			log.Debugf("Domain found, name: " + domainName + "; id: " + domainId)
		} else {
			log.Debugf("Domain NOT found, attempting to create domain: " + domainName)
			foremanDnsProxy := viper.GetString("foreman.dnsproxy")
			foremanDnsProxyResult, err := f.SearchResource("smart_proxies", foremanDnsProxy)
			if err == nil {
				foremanDnsProxyId := strconv.FormatFloat(foremanDnsProxyResult["id"].(float64), 'f', -1, 64)
				log.Debugf("Foreman smart proxy found, name: " + foremanDnsProxy + "; id: " + foremanDnsProxyId)
				domainId, err = foremanCreateDomain(domainName, foremanDnsProxyId)
				if err == nil {
					log.Debugf("Foreman domain created, name: " + domainName + "; id: " + domainId)
				} else {
					log.Errorf("Foreman domain creation failed, name: " + domainName + ", aborting.")
					return
				}
			} else {
				log.Errorf("Foreman smart proxy not found, aborting. Proxy name: " + foremanDnsProxy)
				return
			}
		}
		//Hostgroup
		hostGroupName := viper.GetString("foreman.host.hostgroup")
		if hostGroupName == "" {
			log.Debugf("Hostgroup not found !")
			return
		}
		hostGroup, err := f.SearchResource("hostgroups", hostGroupName)
		var hostGroupId string
		if err == nil {
			hostGroupId = strconv.FormatFloat(hostGroup["id"].(float64), 'f', -1, 64)
			log.Debugf("Hostgroup found, name: " + hostGroupName + "; id: " + hostGroupId)
		} else {
			log.Errorf("Hostgroup doesnt exist !")
			return
		}
		// Organization
		orgLoc := strings.Split(hostGroupName, "/")
		organizationName := orgLoc[0]
		organization, err := f.SearchResource("organizations", organizationName)
		var organizationId string
		if err == nil {
			organizationId = strconv.FormatFloat(organization["id"].(float64), 'f', -1, 64)
			log.Debugf("Organization found, name: " + organizationName + "; id: " + organizationId)
		} else {
			log.Errorf("Organization doesnt exist !")
			return
		}

		// Location
		var locationName string
		locationName = viper.GetString("foreman.host.location")
		if locationName == "" {
			log.Debugf("Location not found in config, trying to set from TLD")
			locationName = hostNameSplit[len(hostNameSplit)-1]
		}
		location, err := f.SearchResource("locations", locationName)
		var locationId string
		if err == nil {
			locationId = strconv.FormatFloat(location["id"].(float64), 'f', -1, 64)
			log.Debugf("Location found, name: " + locationName + "; id: " + locationId)
		} else {
			log.Errorf("Location doesnt exist !")
			return
		}
		// puppetca
		puppetCaName := viper.GetString("puppet.config.ca")
		if hostGroupName == "" {
			log.Debugf("Puppet CA not found !")
			return
		}
		puppetCa, err := f.SearchResource("smart_proxies", puppetCaName)
		var puppetCaId string
		if err == nil {
			puppetCaId = strconv.FormatFloat(puppetCa["id"].(float64), 'f', -1, 64)
			log.Debugf("Puppet CA found, name: " + puppetCaName + "; id: " + puppetCaId)
		} else {
			log.Errorf("Puppet CA doesnt exist !")
			return
		}
		// environment
		puppetEnvironmentName := viper.GetString("puppet.config.environment")
		if puppetEnvironmentName == "" {
			log.Debugf("Puppet Environment not found !")
			return
		}
		puppetEnvironment, err := f.SearchResource("environments", puppetEnvironmentName)
		var puppetEnvironmentId string
		if err == nil {
			puppetEnvironmentId = strconv.FormatFloat(puppetEnvironment["id"].(float64), 'f', -1, 64)
			log.Debugf("Puppet Environment found, name: " + puppetEnvironmentName + "; id: " + puppetEnvironmentId)
		} else {
			log.Errorf("Puppet Environment doesnt exist !")
			return
		}
		// architecture
		var architectureName string
		if puppetVersion == 4 {
			architectureName = viper.GetString("puppetfacter.os.hardware")
		} else {
			architectureName = viper.GetString("puppetfacter.hardwaremodel")
		}
		if architectureName == "" {
			log.Debugf("Architecture not found !")
			return
		}
		architecture, err := f.SearchResource("architectures", architectureName)
		var architectureId string
		if err == nil {
			architectureId = strconv.FormatFloat(architecture["id"].(float64), 'f', -1, 64)
			log.Debugf("Architecture found, name: " + architectureName + "; id: " + architectureId)
		} else {
			log.Errorf("Architecture doesnt exist !")
			return
		}
		// operatingsystem
		var osName string
		if puppetVersion == 4 {
			osName = viper.GetString("puppetfacter.os.name")
		} else {
			osName = viper.GetString("puppetfacter.lsbdistid")
		}
		var operatingSystemName string
		if osName == "Ubuntu" {
			if puppetVersion == 4 {
				operatingSystemName = viper.GetString("puppetfacter.os.distro.description")
			} else {
				operatingSystemName = viper.GetString("puppetfacter.lsbdistdescription")
			}
		} else {
			operatingSystemName = viper.GetString("puppetfacter.os.distro.id") + " " + viper.GetString("puppetfacter.os.distro.release.full")
		}
		if operatingSystemName == "" {
			log.Debugf("Operating System not found !")
			return
		}
		operatingSystem, err := f.SearchResource("operatingsystems", operatingSystemName)
		var operatingSystemId string
		if err == nil {
			operatingSystemId = strconv.FormatFloat(operatingSystem["id"].(float64), 'f', -1, 64)
			log.Debugf("Operating System found, name: " + operatingSystemName + "; id: " + operatingSystemId)
		} else {
			log.Errorf("Operating System doesnt exist !" + operatingSystemName)
			return
		}
		// ipAddress
		if puppetVersion == 4 {
			iface := viper.GetString("facter.interface")
			if iface != "" {
				log.Debugf("Set custom interface to fetch ip from: " + iface)
				ipAddress = viper.GetString("puppetfacter.networking.interfaces." + iface + ".ip")
				if ipAddress == "" {
					log.Debugf("Failed to fetch ip from: " + iface + ", defaulting to puppetfacter.networking.ip")
					ipAddress = viper.GetString("puppetfacter.networking.ip")
				}
			} else {
				ipAddress = viper.GetString("puppetfacter.networking.ip")
			}
		} else {
			ipAddress = viper.GetString("puppetfacter.ipaddress")
		}
		if ipAddress == "" {
			log.Debugf("IP Address not found !")
			return
		} else {
			log.Debugf("IP Address: " + ipAddress)
		}
		// macAddress
		var macAddress string
		if puppetVersion == 4 {
			macAddress = viper.GetString("puppetfacter.networking.mac")
		} else {
			macAddress = viper.GetString("puppetfacter.macaddress")
		}
		if macAddress == "" {
			log.Debugf("Mac Address not found !")
			return
		} else {
			log.Debugf("Mac Address: " + macAddress)
		}
		// parameters
		var parameters []map[string]string
		var paramMap map[string]string
		var tierset bool
		metaparameters, err := metaGetMerge("foreman.host.parameter")
		for metak, metav := range metaparameters {
			paramMap = make(map[string]string)
			paramMap["name"] = metak
			if metak == "tier" {
				tierset = true
			}
			paramMap["value"] = metav
			parameters = append(parameters, paramMap)
		}
		if err != nil {
			log.Debugf("Did not find host parameters")
		}
		// look for tier specificly
		tier := viper.GetString("foreman.host.parameter.tier")
		if tier != "" {
			tierMap := make(map[string]string)
			tierMap["name"] = "tier"
			tierMap["value"] = tier
			if !tierset {
				parameters = append(parameters, tierMap)
			}
			log.Debugf("Set tier: " + tier)
		}
		// basic dns must be handled before host creation due to foreman conflicts
		// Configure DNS
		dnsHost := viper.GetString("dns.config.host")
		if dnsHost == "" {
			log.Debugf("DNS host not configure, skipping")
		} else {
			log.Debugf("Starting DNS record management for host: " + dnsHost)
			dnsKey := viper.GetString("dns.config.key")
			if dnsKey == "" {
				log.Debugf("DNS key not found !")
				return
			}
			// Inicialize powerdns
			dnsNameservers := viper.GetStringSlice("dns.config.nameservers")
			p = powerdns.NewPowerdns(dnsHost, dnsKey, dnsNameservers)
			dnsDeleteRecordHostA()
			dnsRecordHostA()
			dnsRecordHostPtr()
			// Lookup for config values and setup records
			doMetaSliceMap("dns.record.a", dnsRecordA)
			doMetaSliceMap("dns.record.mya", dnsRecordMyA)
			doMetaSliceMap("dns.record.cname", dnsRecordCname)
			doMetaSlice("dns.record.mycname", dnsRecordMyCname)
			doMetaSlice("dns.record.mypubcname", dnsRecordMyPubCname)
			doMetaSliceMap("dns.record.roota", dnsRecordRootA)

		}

		// create host
		type HostResource struct {
			HostGroupId         string              `json:"hostgroup_id"`
			PuppetCaId          string              `json:"puppet_ca_proxy_id"`
			LocationId          string              `json:"location_id"`
			OrganizationId      string              `json:"organization_id"`
			PuppetEnvironmentId string              `json:"environment_id"`
			DomainId            string              `json:"domain_id"`
			OperatingSystemId   string              `json:"operatingsystem_id"`
			ArchitectureId      string              `json:"architecture_id"`
			Name                string              `json:"name"`
			Mac                 string              `json:"mac"`
			Ip                  string              `json:"ip"`
			Build               bool                `json:"build"`
			Parameters          []map[string]string `json:"host_parameters_attributes"`
		}
		type HostMap map[string]HostResource

		//var hostMap map[string]HostResource
		hostMap := make(HostMap)
		hostMap["host"] = HostResource{
			HostGroupId:         hostGroupId,
			PuppetCaId:          puppetCaId,
			LocationId:          locationId,
			OrganizationId:      organizationId,
			PuppetEnvironmentId: puppetEnvironmentId,
			DomainId:            domainId,
			OperatingSystemId:   operatingSystemId,
			ArchitectureId:      architectureId,
			Name:                hostName,
			Mac:                 macAddress,
			Ip:                  ipAddress,
			Build:               false,
			Parameters:          parameters,
		}
		jsonText, err := json.Marshal(hostMap)
		data, err := foremanCreate(jsonText)
		if err != nil {
			log.Errorf("Failed to create host in foreman !")
			return
		}
		hostId := strconv.FormatFloat(data["id"].(float64), 'f', 0, 64)
		log.Debugf("Host created, id: " + hostId)

		// Configure SQL
		doMetaSliceMap("mysql.record", mySqlRecord)
		// Configure Jenkins
		doMetaSliceMap("jenkins.job", jenkinsJob)

		// Configure Puppet execution
		puppetServer := viper.GetString("puppet.config.server")
		var puppetParam []string
		if puppetServer == "" {
			log.Debugf("Puppet Server not found !")
			puppetSrv := viper.GetString("puppet.config.srv")
			if puppetSrv == "" {
				log.Errorf("Puppet Server or SRV not found, exiting !")
				return
			} else {
				log.Debugf("Puppet will run in SRV mode in domain: " + puppetSrv)
				if puppetVersion == 4 {
					puppetParam = []string{"agent", "-tv", "--use_srv_records", "--srv_domain", puppetSrv}
				} else {
					puppetParam = []string{"agent", "-tv", "--use_srv_records", "--srv_domain", puppetSrv, "--pluginsync", "--pluginsource", "puppet:///plugins", "--pluginfactsource", "puppet:///pluginfacts", "--configtimeout", "1200"}
				}
			}
		} else {
			log.Debugf("Puppet will run in server mode with server: " + puppetServer)
			puppetParam = []string{"agent", "-tv", "--no-use_srv_records", "--ca_server", puppetCaName, "--server", puppetServer}
		}
		// Enable Puppet
		log.Debugf("Enabling puppet")
		var puppetExecutable string
		if puppetVersion == 4 {
			puppetExecutable = "/opt/puppetlabs/bin/puppet"
		} else {
			puppetExecutable = "/usr/bin/puppet"
		}
		puppetEnabler := exec.Command(puppetExecutable, "agent", "--enable")
		c := make(chan struct{})
		go runCommand(puppetEnabler, c)
		c <- struct{}{}
		puppetEnabler.Start()
		<-c
		if err := puppetEnabler.Wait(); err != nil {
			log.Debugf("Error executing puppet !")
		}
		// Run Puppet
		puppetRuns := viper.GetInt("puppet.config.runs")
		for r := 1; r <= puppetRuns; r++ {
			// Run puppet
			runCount := strconv.Itoa(r)
			log.Debugf("Running puppet, run #" + runCount)
			//spew.Dump(puppetParam)
			cmd := exec.Command(puppetExecutable, puppetParam...)
			c := make(chan struct{})
			go runCommand(cmd, c)
			c <- struct{}{}
			cmd.Start()
			<-c
			if err := cmd.Wait(); err != nil {
				log.Debugf("Error executing puppet !")
				spew.Dump(err)
				if puppetSslError {
					log.Debugf("Puppet SSL Error detected !")
					foremanDelete(hostFqdn)
					data, err := foremanCreate(jsonText)
					if err != nil {
						log.Errorf("Failed to create host in foreman !")
						return
					}
					hostId := strconv.FormatFloat(data["id"].(float64), 'f', 0, 64)
					log.Debugf("Host created, id: " + hostId)
					var puppetSsl string
					if puppetVersion == 4 {
						puppetSsl = "/etc/puppetlabs/puppet/ssl"
					} else {
						puppetSsl = "/var/lib/puppet/ssl"
					}
					puppetSslFix := exec.Command("rm", "-rf", puppetSsl)
					s := make(chan struct{})
					go runCommand(puppetSslFix, s)
					s <- struct{}{}
					puppetSslFix.Start()
					<-s
					if err := puppetSslFix.Wait(); err != nil {
						log.Debugf("Error deleting Puppet SSL dir !")
					}
				}
			}
		}
	},
}

func runCommand(cmd *exec.Cmd, c chan struct{}) {
	puppetSslError = false
	defer func() { c <- struct{}{} }()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	<-c
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}
	errScanner := bufio.NewScanner(stderr)
	for errScanner.Scan() {
		e := errScanner.Text()
		fmt.Println(e)
		if strings.Contains(e, "The certificate retrieved from the master does not match the agent") {
			puppetSslError = true
			log.Debugf("SSL Error:" + e)
		}
	}
}

func init() {
	RootCmd.AddCommand(createCmd)
}

func doMetaSliceMap(config string, f func(map[string]interface{})) {
	for i := 0; i < 100; i++ {
		var lookup string
		if i == 0 {
			lookup = config
		} else {
			lookupindex := strconv.Itoa(i)
			lookup = config + lookupindex
		}
		iface := viper.Get(lookup)
		if iface != nil {
			islice, ok := iface.([]interface{})
			if ok {
				for _, islicev := range islice {
					islicemap, ok := islicev.(map[string]interface{})
					if ok {
						f(islicemap)
					} else {
						log.Debugf("Array value in " + config + " is not a Hash !")
					}
				}
			} else {
				log.Debugf("Record " + config + " is not Array!")
			}
		}
	}
}

func doMetaSlice(config string, f func(string)) {
	for i := 0; i < 100; i++ {
		var lookup string
		if i == 0 {
			lookup = config
		} else {
			lookupindex := strconv.Itoa(i)
			lookup = config + lookupindex
		}
		iface := viper.Get(lookup)
		if iface != nil {
			islice, ok := iface.([]interface{})
			if ok {
				for _, islicev := range islice {
					islicestring, ok := islicev.(string)
					if ok {
						f(islicestring)
					} else {
						log.Debugf("Array value in " + config + " is not a String !")
					}
				}
			} else {
				log.Debugf("Record " + config + " is not Array!")
			}
		}
	}
}

func dnsRecordHostA() {
	err := p.UpdateRecord(domainName, "A", hostName, ipAddress, 60)
	if err != nil {
		log.Debugf("Failed to update A record, domain: " + domainName + ", content: " + hostName + ", value: " + ipAddress + " !")
	}
	log.Debugf("Updated A record, domain: " + domainName + ", content: " + hostName + ", value: " + ipAddress + " !")
}

func dnsDeleteRecordHostA() {
	err := p.DeleteRecord(domainName, "A", hostName)
	if err != nil {
		log.Debugf("Failed to delete A record, domain: " + domainName + ", content: " + hostName + " !")
	}
	log.Debugf("Deleted A record, domain: " + domainName + ", content: " + hostName + " !")
}

func dnsRecordHostPtr() {
	ipAddressSlice := strings.Split(ipAddress, ".")
	ptrRecord := ipAddressSlice[3] + "." + ipAddressSlice[2] + "." + ipAddressSlice[1] + "." + ipAddressSlice[0] + ".in-addr.arpa."
	ptrDomain := ipAddressSlice[2] + "." + ipAddressSlice[1] + "." + ipAddressSlice[0] + ".in-addr.arpa"
	err := p.UpdateRec(ptrDomain, "PTR", ptrRecord, hostFqdn+".", 60)
	if err != nil {
		log.Debugf("Failed to update PTR record, domain: " + ptrDomain + ", content: " + ptrRecord + ", value: " + hostFqdn + " !")
	}
	log.Debugf("Updated PTR record, domain: " + ptrDomain + ", content: " + ptrRecord + ", value: " + hostFqdn + " !")
}

func dnsRecordMyA(hash map[string]interface{}) {
	for k, v := range hash {
		pK, err := metaTemplate(k)
		if err != nil {
			log.Debugf("Failed to parse dns.record.a key " + k + " !")
			return
		}
		pV, err := metaTemplate(v.(string))
		if err != nil {
			log.Debugf("Failed to parse dns.record.a value " + v.(string) + " !")
			return
		}
		err = p.UpdateRecord(domainName, "A", pK, pV, 60)
		if err != nil {
			log.Debugf("Failed to update A record, domain: " + domainName + ", content: " + pK + ", value: " + pV + " !")
		}
		log.Debugf("Updated A record, domain: " + domainName + ", content: " + pK + ", value: " + pV + " !")
	}
}

func dnsRecordA(hash map[string]interface{}) {
	for k, v := range hash {
		pK, err := metaTemplate(k)
		if err != nil {
			log.Debugf("Failed to parse dns.record.a key " + k + " !")
			return
		}
		pV, err := metaTemplate(v.(string))
		if err != nil {
			log.Debugf("Failed to parse dns.record.a value " + v.(string) + " !")
			return
		}

		pKSplit := strings.Split(pK, ".")
		pKHostName := pKSplit[0]
		pKDomainName := strings.Replace(pK, pKHostName+".", "", -1)

		err = p.UpdateRecord(pKDomainName, "A", pKHostName, pV, 60)
		if err != nil {
			log.Debugf("Failed to update A record, domain: " + pKDomainName + ", content: " + pKHostName + ", value: " + pV + " !")
		}
		log.Debugf("Updated A record, domain: " + pKDomainName + ", content: " + pKHostName + ", value: " + pV + " !")
	}
}

func dnsRecordRootA(hash map[string]interface{}) {
	for k, v := range hash {
		pK, err := metaTemplate(k)
		if err != nil {
			log.Debugf("Failed to parse dns.record.a key " + k + " !")
			return
		}
		pV, err := metaTemplate(v.(string))
		if err != nil {
			log.Debugf("Failed to parse dns.record.a value " + v.(string) + " !")
			return
		}

		//pKSplit := strings.Split(pK, ".")
		//pKHostName := pKSplit[0]
		//pKDomainName := strings.Replace(pK, pKHostName+".", "", -1)
		pKDomainName := pK + "."
		pKHostName := pK + "."
		err = p.UpdateRec(pKDomainName, "A", pKHostName, pV, 60)
		if err != nil {
			log.Debugf("Failed to update Root A record, domain: " + pKDomainName + ", content: " + pKHostName + ", value: " + pV + " !")
		}
		log.Debugf("Updated Root A record, domain: " + pKDomainName + ", content: " + pKHostName + ", value: " + pV + " !")
	}
}

func dnsRecordCname(hash map[string]interface{}) {
	for k, v := range hash {
		pK, err := metaTemplate(k)
		if err != nil {
			log.Debugf("Failed to parse dns.record.cname key " + k + " !")
			return
		}
		pV, err := metaTemplate(v.(string))
		if err != nil {
			log.Debugf("Failed to parse dns.record.cname value " + v.(string) + " !")
			return
		}

		pKSplit := strings.Split(pK, ".")
		pKHostName := pKSplit[0]
		pKDomainName := strings.Replace(pK, pKHostName+".", "", -1)

		err = p.UpdateRecord(pKDomainName, "CNAME", pKHostName, pV+".", 60)
		if err != nil {
			log.Debugf("Failed to update CNAME record, domain: " + pKDomainName + ", content: " + pKHostName + ", value: " + pV + ". !")
		}
		log.Debugf("Updated CNAME record, domain: " + pKDomainName + ", content: " + pKHostName + ", value: " + pV + ". !")
	}
}

func dnsRecordMyPubCname(s string) {
	pS, err := metaTemplate(s)
	if err != nil {
		log.Debugf("Failed to parse dns.record.mypubcname value " + s + " !")
		return
	}
	pSSplit := strings.Split(pS, ".")
	pSHostName := pSSplit[0]
	pSDomainName := strings.Replace(pS, pSHostName+".", "", -1)
	err = p.UpdateRecord(pSDomainName, "CNAME", pSHostName, hostFqdn+".", 60)
	if err != nil {
		log.Debugf("Failed to update CNAME record, domain: " + pSDomainName + ", content: " + pSHostName + ", value: " + hostFqdn + ". !")
	}
	log.Debugf("Updated CNAME record, domain: " + pSDomainName + ", content: " + pSHostName + ", value: " + hostFqdn + ". !")
}

func dnsRecordMyCname(s string) {
	pS, err := metaTemplate(s)
	if err != nil {
		log.Debugf("Failed to parse dns.record.mycname value " + s + " !")
		return
	}
	err = p.UpdateRecord(domainName, "CNAME", pS, hostFqdn+".", 60)
	if err != nil {
		log.Debugf("Failed to update CNAME record, domain: " + domainName + ", content: " + pS + ", value: " + hostFqdn + ". !")
	}
	log.Debugf("Updated CNAME record, domain: " + domainName + ", content: " + pS + ", value: " + hostFqdn + ". !")
}

func mySqlRecord(hash map[string]interface{}) {
	uri := hash["uri"].(string)
	if len(uri) == 0 {
		log.Errorf("URI empty in mysql.record !")
		return
	}
	uriSplit := strings.Split(uri, ".")
	db := uriSplit[0]
	table := uriSplit[1]
	dbHostRaw := viper.GetString("mysql.db." + db + ".host")
	if dbHostRaw == "" {
		log.Errorf("DB Host mysql.db." + db + ".host not found in config !")
		return
	}
	dbHost, err := metaTemplate(dbHostRaw)
	if err != nil {
		log.Errorf("DB Host value " + dbHostRaw + "failed to be parsed !")
		return
	}
	dbUser := viper.GetString("mysql.db." + db + ".user")
	if dbUser == "" {
		log.Errorf("DB User mysql.db." + db + ".user not found in config !")
		return
	}
	dbPassword := viper.GetString("mysql.db." + db + ".password")
	if dbPassword == "" {
		log.Errorf("DB Password mysql.db." + db + ".password not found in config !")
		return
	}
	template := hash["template"].(string)
	if len(template) == 0 {
		log.Errorf("Template empty in mysql.record !")
		return
	}
	data := viper.Get(template).(map[string]interface{})
	if data == nil {
		log.Errorf("Template data " + template + " not found in config !")
		return
	}
	dataLength := len(data)
	var values []interface{}
	index := 1
	var keys string
	var questions string
	var valuesString string
	for k, v := range data {
		vC := fmt.Sprintf("%v", v)
		vS, err := metaTemplate(vC)
		if err != nil {
			log.Debugf("Failed to parse " + template + " value " + vC + " !")
			return
		}
		var vSI interface{} = vS
		values = append(values, vSI)
		if index < dataLength {
			keys = keys + k + ","
			questions = questions + "?,"
			valuesString = valuesString + vS + ","
		} else {
			keys = keys + k
			questions = questions + "?"
			valuesString = valuesString + vS
		}
		index++
	}
	// Create an sql.DB and check for errors
	//var err error
	d, err = sql.Open("mysql", dbUser+":"+dbPassword+"@tcp("+dbHost+":3306)/"+db)
	if err != nil {
		log.Errorf("Database open failed: " + err.Error())
		return
	}
	defer d.Close()
	err = d.Ping()
	if err != nil {
		log.Errorf("Database connection failed: " + err.Error())
		return
	}
	_, err = d.Exec("INSERT INTO "+table+" ("+keys+") VALUES("+questions+")", values...)
	if err != nil {
		log.Errorf("Error inserting record into database: " + err.Error())
		return
	}
	log.Debugf("Sucessfully inserted SQL record into mysql database " + dbUser + ":<PASS DEDACTED>@tcp(" + dbHost + ":3306)/" + db + " : INSERT INTO " + table + " (" + keys + ") VALUES(" + valuesString + ")")
}

func jenkinsJob(hash map[string]interface{}) {
	uri := hash["uri"].(string)
	if len(uri) == 0 {
		log.Errorf("URI empty in jenkins.job !")
		return
	}
	jenkinsHost := viper.GetString("jenkins.host." + uri + ".host")
	if jenkinsHost == "" {
		log.Errorf("Jenkins host jenkins.host." + uri + ".host not found in config !")
		return
	}
	jenkinsUser := viper.GetString("jenkins.host." + uri + ".user")
	if jenkinsUser == "" {
		log.Errorf("Jenkins user jenkins.host." + uri + ".user not found in config !")
		return
	}
	jenkinsPassword := viper.GetString("jenkins.host." + uri + ".password")
	if jenkinsPassword == "" {
		log.Errorf("Jenkins password jenkins.host." + uri + ".password not found in config !")
		return
	}
	templateFile := hash["template"].(string)
	if len(templateFile) == 0 {
		log.Errorf("Template empty in jenkins.job !")
		return
	}
	nameSource := hash["name"].(string)
	if len(nameSource) == 0 {
		log.Errorf("Name empty in jenkins.job !")
		return
	}
	name, err := metaTemplate(nameSource)
	if err != nil {
		log.Debugf("Failed to parse name value " + nameSource + " !")
		return
	}
	templateFileContent, err := ioutil.ReadFile(templateFile) // just pass the file name
	if err != nil {
		log.Debugf("Failed to open Jenkins template file: " + templateFile + " !")
		return
	}
	templateStringContent := string(templateFileContent) // convert content to a 'string'
	jobXml, err := metaTemplate(templateStringContent)
	if err != nil {
		log.Debugf("Failed to parse Jenkins template file " + templateFile + " with content: " + templateStringContent)
		return
	}
	// Inicialize jenkins
	j = jenkins.NewJenkins(jenkinsHost, jenkinsUser, jenkinsPassword)

	//	jobXmlBytes := []byte(jobXml)
	job := html.UnescapeString(jobXml)
	projectName := html.EscapeString(name)
	response, err := j.Post("createItem?name="+projectName, job)
	if err != nil {
		log.Errorf("Error creating host !")
		return
	}
	spew.Dump(response)
	// Create an sql.DB and check for errors
	//	var err error
	//	d, err = sql.Open("mysql", dbUser+":"+dbPassword+"@tcp("+dbHost+":3306)/"+db)
	//	if err != nil {
	//		log.Errorf("Database open failed: " + err.Error())
	//		return
	//	}
	//	defer d.Close()
	//	err = d.Ping()
	//	if err != nil {
	//		log.Errorf("Database connection failed: " + err.Error())
	//		return
	//	}
	//	_, err = d.Exec("INSERT INTO "+table+" ("+keys+") VALUES("+questions+")", values...)
	//	if err != nil {
	//		log.Errorf("Error inserting record into database: " + err.Error())
	//		return
	//	}
	//	log.Debugf("Sucessfully inserted SQL record into mysql database " + dbUser + ":<PASS DEDACTED>@tcp(" + dbHost + ":3306)/" + db + " : INSERT INTO " + table + " (" + keys + ") VALUES(" + valuesString + ")")
}

func foremanCreate(jsonText []byte) (map[string]interface{}, error) {
	data, err := f.Post("hosts", jsonText)
	if err != nil {
		log.Errorf("Error creating host, retrying in 5s !")
		time.Sleep(5 * time.Second)
		data, err = f.Post("hosts", jsonText)
		if err != nil {
			log.Errorf("Error creating host, retrying in 15s !")
			time.Sleep(15 * time.Second)
			data, err = f.Post("hosts", jsonText)
			if err != nil {
				log.Errorf("Error creating host, retrying in 60s !")
				for i := 1; i < 31; i++ {
					time.Sleep(60 * time.Second)
					data, err = f.Post("hosts", jsonText)
					if err != nil {
						log.Errorf("Error creating host, retrying in 60s, cycle !")
					} else {
						return data, err
					}
				}
				log.Errorf("Error creating host, giving up !")
				return nil, err
			}
		}
	}
	return data, err
}

func foremanDelete(hostFqdn string) error {
	host, err := f.SearchResource("hosts", hostFqdn)
	if err == nil {
		log.Debugf("Host exists, deleting")
		hostId := strconv.FormatFloat(host["id"].(float64), 'f', -1, 64)
		err := f.DeleteHost(hostId)
		if err != nil {
			log.Errorf("Error deleting host, retrying in 5s !")
			time.Sleep(5 * time.Second)
			err := f.DeleteHost(hostId)
			if err != nil {
				log.Errorf("Error deleting host, retrying in 15s !")
				time.Sleep(15 * time.Second)
				err := f.DeleteHost(hostId)
				if err != nil {
					log.Errorf("Error deleting host, retrying in 60s !")
					for i := 1; i < 31; i++ {
						time.Sleep(60 * time.Second)
						err := f.DeleteHost(hostId)
						if err != nil {
							log.Errorf("Error deleting host, retrying in 60s !")
						} else {
							return err
						}
					}
					log.Errorf("Error deleting host, giving up !")
					return err
				}
			}
			return err
		}
	}
	return err
}

func foremanCreateResource(jsonText []byte, resource string) (map[string]interface{}, error) {
	data, err := f.Post(resource, jsonText)
	if err != nil {
		log.Errorf("Error creating resource: " + resource + ", retrying in 5s !")
		time.Sleep(5 * time.Second)
		data, err = f.Post(resource, jsonText)
		if err != nil {
			log.Errorf("Error creating resource: " + resource + ", retrying in 15s !")
			time.Sleep(15 * time.Second)
			data, err = f.Post(resource, jsonText)
			if err != nil {
				log.Errorf("Error creating resource: " + resource + ", retrying in 60s !")
				for i := 1; i < 31; i++ {
					time.Sleep(60 * time.Second)
					data, err = f.Post(resource, jsonText)
					if err != nil {
						log.Errorf("Error creating resource: " + resource + ", retrying in 60s, cycle !")
					} else {
						return data, err
					}
				}
				log.Errorf("Error creating resource: " + resource + ", giving up !")
				return nil, err
			}
		}
	}
	return data, err
}

func foremanCreateDomain(domain string, foremanProxy string) (domain_id string, err error) {
	// create domain
	type DomainResource struct {
		Name  string `json:"name"`
		DnsId string `json:"dns_id"`
	}
	type DomainMap map[string]DomainResource

	//var hostMap map[string]HostResource
	domainMap := make(DomainMap)
	domainMap["domain"] = DomainResource{
		Name:  domain,
		DnsId: foremanProxy,
	}
	jsonText, err := json.Marshal(domainMap)
	data, err := foremanCreateResource(jsonText, "domains")
	if err != nil {
		log.Errorf("Failed to create domain in foreman !")
		return "", err
	}
	domainId := strconv.FormatFloat(data["id"].(float64), 'f', 0, 64)
	log.Debugf("Domain created, id: " + domainId)
	return domainId, nil
}
