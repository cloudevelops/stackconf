# Install development environment

## setup git

```
sudo apt-get install -y git
git config --global user.email "zdenek.janda@cloudevelops.com"
git config --global user.name "Zdenek Janda"
```

## setup golang

```
wget -O /tmp/go1.8.3.linux-amd64.tar.gz https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf /tmp/go1.8.3.linux-amd64.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin" >> ~/.bashrc
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
echo "export GOPATH=$HOME/go" >> ~/.bashrc
export GOPATH=$HOME/go
```

## get stackconf

```
mkdir -p ~/go
go get github.com/cloudevelops/stackconf
```

## setup fpm for packaging

```
sudo apt-get install ruby-dev -y
sudo gem install fpm
```

## optionally, setup go-enabled vim

```
sudo apt-get install -y vim-nox
curl https://j.mp/spf13-vim3 -L -o - | sh
```

# Build package

## Edit a version in debian/package.sh

```
fpm -s dir -t deb -C /tmp/stackconf --name stackconf --version 0.1.10 --description "stack orchestration engine" --package /tmp/stackconf
```

## Execute debian/package.sh, new version is built in /tmp. Scp it to apt.cloudevelops.com:

```
scp /tmp/stackconf...deb apt.cloudevelops.com:/tmp/
```

## on apt.cloudevelops.com, execute install oneliner to deploy stackconf

```
STACKCONF=/tmp/stackconf_0.1.19_amd64.deb; aptly repo add cloudevelops-xenial $STACKCONF; aptly repo add cloudevelops-stretch $STACKCONF; aptly repo add cloudevelops-trusty $STACKCONF; aptly repo add cloudevelops-bionic $STACKCONF; aptly repo add cloudevelops-jessie $STACKCONF; aptly publish update xenial cloudevelops && aptly publish update jessie cloudevelops && aptly publish update trusty cloudevelops && aptly publish update bionic cloudevelops && aptly publish update stretch cloudevelops && aptly repo add cloudevelops-buster $STACKCONF && aptly publish update buster cloudevelops && aptly repo add cloudevelops-focal $STACKCONF && aptly publish update focal cloudevelops
```
