#!/usr/local/node/bin/node
var Web3 = require('web3');
var web3 = new Web3(new Web3.providers.HttpProvider("http://54.252.195.103:8545"));//节点地址
var abi = require("./abi.json");
var address = "0x84fc41Ee42872c7eE511025dCbC00E32cdA6b079";  //合约地址
var contract = new web3.eth.Contract(abi,address); //合约实例
 var from='0xd3a4394d69f7ba85544c8ce7e5d2f8aa57de18f3'; //部署账号
 var to="0x0d7c521218364d549e69ff48945175106711ba70";
 contract.methods.mint(to,web3.utils.toWei("210000000","ether")).send({from:from})
		  .on('transactionHash', function(hash){
		  console.log(hash);
		  })
		  .on('confirmation', function(confirmationNumber, receipt){

		  })
		  .on('receipt', function(receipt){

          console.log(receipt);
		  
		  });