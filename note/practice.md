CentOS用練環境構築

## SELinux無効化
```
sudo vim /etc/selinux/config
# SELINUX=disabled
sudo reboot
```

## パスワードなしsudo
```
sudo visudo
# %group        ALL=(ALL)       NOPASSWD: ALL
isucon        ALL=(ALL)       NOPASSWD: ALL
```

## Systemd
```sh
sudo systemctl daemon-reload

sudo systemctl enable nginx
sudo systemctl disable nginx

sudo systemctl restart nginx
sudo systemctl start nginx
sudo systemctl stop nginx

sudo systemctl status nginx

sudo journalctl -u nginx
sudo journalctl -f -u nginx

sudo systemctl list-unit-files -t service
```

## isuconユーザ作成
```sh
sudo yum install -y curl
sudo useradd isucon
sudo passwd isucon
sudo usermod -G wheel isucon

sudo mkdir -p /home/isucon/.ssh
curl https://github.com/inada-s.keys | sudo tee -a /home/isucon/.ssh/authorized_keys
sudo chown -R isucon:isucon /home/isucon/.ssh
sudo chmod 700 /home/isucon/.ssh
sudo chmod 600 /home/isucon/.ssh/authorized_keys

sudo yum -y update
sudo reboot
```

## Moromoro
```sh
sudo yum groupinstall -y "Development Tools"
sudo yum install -y kernel-devel kernel-headers
sudo yum install -y git curl screen wget tmux git tree make python-dev python-pip python3-dev python3-pip unzip zip graphviz sysstat
```

## FlameGraph
```sh
wget https://github.com/brendangregg/FlameGraph/archive/master.zip
unzip master.zip
sudo mv FlameGraph-master/* /usr/local/bin/
rm -r FlameGraph-master
rm master.zip
```

## Golang
```sh
wget https://dl.google.com/go/go1.11.linux-amd64.tar.gz && sudo tar -C /usr/local -xzf go1.11.linux-amd64.tar.gz && rm go1.11.linux-amd64.tar.gz
GOPATH=~/go
GOBIN=~/go/bin
mkdir -p ~/go
cat << EOF >> ~/.bash_profile
export PATH=$PATH:/usr/local/go/bin:$GOBIN
export GOPATH=$GOPATH
EOF
source ~/.bash_profile
```

## Vim
```sh
sudo yum -y remove vim-enhanced
sudo yum -y install lua-devel ncurses-devel
sudo yum -y install ruby ruby-devel lua lua-devel luajit luajit-devel ctags mercurial python python-devel python3 python3-devel tcl-devel perl perl-devel perl-ExtUtils-ParseXS perl-ExtUtils-XSpp perl-ExtUtils-CBuilder perl-ExtUtils-Embed ncurses-devel
git clone https://github.com/vim/vim.git
cd vim
./configure --with-features=huge --enable-multibyte --enable-rubyinterp --enable-pythoninterp --enable-perlinterp --enable-luainterp --enable-gui=gtk2 --enable-cscope --with-tlib=ncurses --prefix=/usr
sudo make install

cd ~
curl https://raw.githubusercontent.com/Shougo/neobundle.vim/master/bin/install.sh > neobundle-install.sh
sh ./neobundle-install.sh
rm neobundle-install.sh

curl https://raw.githubusercontent.com/inada-s/isucon/master/.vimrc > ~/.vimrc
vim +NeoBundleInstall  +qall
vim +GoInstallBinaries +qall
```

## MySQL
```sh
wget http://repo.mysql.com/mysql-community-release-el7-5.noarch.rpm
sudo rpm -ivh mysql-community-release-el7-5.noarch.rpm
sudo yum -y update
sudo yum -y install mysql-server
sudo systemctl start mysqld
sudo systemctl enable mysqld
```

## MySQL設定
```sql
CREATE USER 'isucon'@'localhost' IDENTIFIED BY 'isucon';
GRANT ALL PRIVILEGES ON *.* TO 'isucon'@'localhost';
FLUSH PRIVILEGES;

```

古い認証方式に戻す
```sql
ALTER USER 'isucon'@'localhost' IDENTIFIED WITH mysql_native_password BY 'isucon';
```

```
[mysqld]
default-authentication-plugin = mysql_native_password
```

## myprofiler
```sh
wget https://github.com/KLab/myprofiler/releases/download/0.1/myprofiler.linux_amd64.tar.gz
tar xf myprofiler.linux_amd64.tar.gz
sudo mv myprofiler /usr/local/bin/
```

## Nginx
```
sudo yum -y install epel-release
sudo yum -y install nginx
sudo systemctl enable nginx
sudo systemctl start nginx
```

## Memcache
```sh
sudo yum -y install memcached
#sudo vim /etc/sysconfig/memcached
sudo systemctl enable memcached
sudo systemctl start memcached
```


## Firewall
多分不要
```sh
sudo firewall-cmd --permanent --zone=public --add-service=http 
sudo firewall-cmd --permanent --zone=public --add-service=https
sudo firewall-cmd --reload
```


## pprof

```sh
for isu in $ISUS; do
  ssh $isu 'sudo yum install -y dstat sysstat'
done
```

```sh
scp on-start-profile $isu:on-start-profile
```

pprof.goを貼り付けて make deploy restart
ログを見ながら
```sh
curl isu1/startprof
curl isu1/endprof
```
を叩いて正しくプロファイル関係のファイルが収集できてるか確認


