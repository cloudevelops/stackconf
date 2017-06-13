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

## optionally, setup go-enabled vim

```
sudo apt-get install -y vim-nox
curl https://j.mp/spf13-vim3 -L -o - | sh
```
