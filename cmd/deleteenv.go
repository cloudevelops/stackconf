// Copyright © 2018 Zdenek Janda <zdenek.janda@cloudevelops.com>
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
	"github.com/cloudevelops/go-foreman"
	"github.com/cloudevelops/go-powerdns"
	//	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strconv"
	"time"
)

var fo *foreman.Foreman

// deleteenvCmd represents the deleteenv command
var deleteenvCmd = &cobra.Command{
	Use:   "deleteenv",
	Short: "Deletes environment",
	Long:  `Deletes environment set by arguments. More environments supported, it will try to find all resources including any present environment string`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			log.Debugf("deleteenv requires at least one environment name to delete")
			return
		}
		log.Debugf("Starting deleteenv")
		//Foreman prototype
		fo = foreman.NewForeman(viper.GetString("foreman.config.host"), viper.GetString("foreman.config.username"), viper.GetString("foreman.config.password"))
		// Host
		for _, env := range args {
			//hostNameSplit := strings.Split(hostFqdn, ".")
			//hostName = hostNameSplit[0]
			//domainName = strings.Replace(hostFqdn, hostName+".", "", -1)
			data, err := fo.SearchAnyResource("hosts", env)
			if err == nil {
				resultSlice := data["results"].([]interface{})
				for _, resultItem := range resultSlice {
					resultData := resultItem.(map[string]interface{})
					//resultData := resultItem
					hostName = resultData["name"].(string)
					log.Debugf("Deleting host: " + hostName)
					//				if title, ok := resultData["title"]; ok {
					//				if title == Query {
					//				return resultData, err
					//		}
					//}
					//}
					//			spew.Dump(host)
					//			hostId := strconv.FormatFloat(host["id"].(float64), 'f', -1, 64)
					//			err := f.DeleteHost(hostId)
					//			if err != nil {
					//				log.Debugf("Host deletion failed")
					err := foremanDel(hostName)
					if err != nil {
						log.Debugf("Foreman failed to delete host: " + hostName + " !")
					}
				}
			}
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
				p = powerdns.NewPowerdns(dnsHost, dnsKey)
				querydomain := env
				domain, err := p.Get("zones/" + querydomain)
				if err == nil {
					domainMap := domain.(map[string]interface{})
					rrsets := domainMap["rrsets"].([]interface{})
					for _, rrset := range rrsets {
						rrdata := rrset.(map[string]interface{})
						rrtype := rrdata["type"].(string)
						rrname := rrdata["name"].(string)
						if rrtype == "A" {
							err := p.DeleteRec(querydomain, rrtype, rrname)
							if err != nil {
								log.Debugf("Failed to delete " + rrtype + " record, domain: " + querydomain + ", name: " + rrname + " !")
							}
							log.Debugf("Deleted " + rrtype + " record, domain: " + domainName + ", name: " + rrname + " !")
						}
						if rrtype == "CNAME" {
							err := p.DeleteRec(querydomain, rrtype, rrname)
							if err != nil {
								log.Debugf("Failed to delete " + rrtype + " record, domain: " + querydomain + ", name: " + rrname + " !")
							}
							log.Debugf("Deleted " + rrtype + " record, domain: " + domainName + ", name: " + rrname + " !")
						}
					}

					//spew.Dump(domainMap)
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(deleteenvCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deleteenvCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deleteenvCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func foremanDel(hostFqdn string) error {
	host, err := fo.SearchResource("hosts", hostFqdn)
	if err == nil {
		log.Debugf("Host exists, deleting")
		hostId := strconv.FormatFloat(host["id"].(float64), 'f', -1, 64)
		err := fo.DeleteHost(hostId)
		if err != nil {
			log.Errorf("Error deleting host, retrying in 5s !")
			time.Sleep(5 * time.Second)
			err := fo.DeleteHost(hostId)
			if err != nil {
				log.Errorf("Error deleting host, retrying in 15s !")
				time.Sleep(15 * time.Second)
				err := fo.DeleteHost(hostId)
				if err != nil {
					log.Errorf("Error deleting host, retrying in 60s !")
					for i := 1; i < 31; i++ {
						time.Sleep(60 * time.Second)
						err := fo.DeleteHost(hostId)
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
