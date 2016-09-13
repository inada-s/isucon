# INSATALL tools
sudo apt-get update
sudo apt-get install -f -y git vim curl wget tmux git tree make python-dev python-pip python3-dev python3-pip unzip zip graphviz 

# INSTALL FlameGraph
wget https://github.com/brendangregg/FlameGraph/archive/master.zip
unzip master.zip
sudo mv FlameGraph-master/* /usr/local/bin/
rm -rf FlameGraph-master

# INSTALL golang
wget https://storage.googleapis.com/golang/go1.7.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.7.linux-amd64.tar.gz
rm go1.7.1.linux-amd64.tar.gz

GOPATH=~/go
GOBIN=~/go/bin
mkdir -p ~/go
cat << EOF >> ~/.profile
export PATH=$PATH:/usr/local/go/bin:$GOBIN
export GOPATH=$GOPATH
EOF

# SETUP vim
curl https://raw.githubusercontent.com/Shougo/neobundle.vim/master/bin/install.sh > neobundle-install.sh
sh ./neobundle-install.sh
rm neobundle-install.sh

curl https://raw.githubusercontent.com/inada-s/isucon/master/.vimrc > ~/.vimrc

source ~/.profile
vim +NeoBundleInstall  +qall
vim +GoInstallBinaries +qall
