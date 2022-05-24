#!/usr/local/node/bin/node
var Web3 = require('web3');
var web3 = new Web3(new Web3.providers.HttpProvider("http://54.252.195.103:8545"));//节点地址
var abi = require("./abi.json");
var address = "0x84fc41Ee42872c7eE511025dCbC00E32cdA6b079";  //合约地址
var contract = new web3.eth.Contract(abi,address); //合约实例
var account="0x0d7c521218364d549e69ff48945175106711ba70";//待查账号
 contract.methods.balanceOf(account).call().then(function (result){
 	console.log(result);
 });
		  