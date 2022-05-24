#!/usr/local/node/bin/node
var Web3 = require('web3');
var web3 = new Web3(new Web3.providers.HttpProvider("http://18.216.66.9:8545"));
console.log(web3.eth.accounts.privateKeyToAccount("0x7be017c2065bd74b58a7260deb631417ef46018b969829499356778a6e39a675"))
