# Stackconf

Stackconf is using basic configuration file stackconf.yaml with variable defaults, which are deployed by heat to target heat stack machines. Environment specific variables can be also overriden in heat environment file.

## Basic stackconf configuration file stackconf.yaml

Stackconf variables in stackconf.yaml are listed in content area and deployed onto target system in /etc/.stackconf.yaml. Content area first level variables are lowest priority defaults. Stackconf can also support multiple
environment defaults, which are in env section.

### Stackconf example:

```
#cloud-config
merge_how: dict(recurse_array)+list(append)
write_files:
  - path: /etc/.stackconf.yaml
    owner: "root:root"
    permissions: "0644"
    content: |
      puppet.config.srv: infra.lan
      puppet.config.ca: puppetca-2.infra.lan
      puppet.config.environment: dev
      foreman.config.username: foreman
      foreman.config.password: rgWptAQcDn4BxMti
      foreman.config.host: foreman.infra.lan
      dns.config.host: dnsmaster-1.infra.lan
      dns.config.key: SomeSecretKey
      env:
        infra_lan:
          foreman.host.parameter.tier: prod
          puppet.config.environment: production
          dns.config.host: dnsmaster-1.infra.lan
          dns.config.key: SomeAnotherKey
```

### Supported stackconf variables

* puppet.config.srv - srv domain which is used for puppet run
* puppet.config.ca - ca server which is uded for puppet run
* puppet.config.environment - environment for puppet run
* puppet.config.server - specific puppet server to use, has priority over puppet.config.srv
* foreman.config.username - username for foreman access
* foreman.config.password - password for foreman access
* foreman.config.host - host used for foreman access
* foreman.host.parameter.[parameter] - value of specific parameter to set for host in foreman
* foreman.host.location - location to set for host in foreman
* dns.config.host - host used for powerdns access
* dns.config.key - key used for powerdns access


## Heat environment files

Basic heat environment file is consisting of parameters, which are used for heat template. Stackconf related variables are in metadata section. It supports all standart stackconf variables, plus a special variable stackenv,
which is selecting specific stackconf environment variables default

### Heat environment file example:

```
parameters:
  project: devel5
  domain: devel5.lan
  ips: 10.5.5
  metadata:
    stackenv: infra_cis
    foreman.host.parameter.appenv: devel5
    foreman.host.location: cis
    puppet.config.environment: devel
```

### Heat environment quick hints

#### Puppetize whole environment againts specific puppetserver

```
parameters:
  metadata:
    foreman.host.parameter.puppetserver: deu-puppetserver1.sandbox.lan
    puppet.config.server: deu-puppetserver1.sandbox.lan
```

#### Select a stackenv environment other than default

```
parameters:
  metadata:
    stackenv: infra_lan
```

#### Select specific puppet environment for stackenv

```
parameters:
  metadata:
    puppet.config.environment: devel
```

# Using stackconf

## Create host

stackconf host create is invoked on target host by cloud init, packer or manually. create subcommand iwll create host and all foreman/dns/etc records in all available APIs managed by stackconf and runs puppet:

```
stackconf create
```

## Delete host

stackconf host delete is invoked by build process or manual step. delete subcommand will remove all host related foreman/dns/etc records in all available APIs managed by stackconf:
```
stackconf delete
```

## Delete environment

environment purging by stack.conf. use one or more environments, distuinquished by dns. This will purge all foreman and DNS A/CNAME records for target environment:
```
stackconf deleteenv dev5.lan dev5.pub
```

# Developing stackconf

## Source code

Source code can be found at https://github.com/cloudevelops/stackconf . Contribute changes via PR on Github.

## Developing, and building package

https://github.com/cloudevelops/stackconf/blob/master/INSTALL.md

