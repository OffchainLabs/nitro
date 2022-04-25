import {ethers} from "hardhat";
import {expect} from "chai";
import TestCase from "./withdraw-testcase.json";
import { BigNumber, Contract } from "ethers";

async function sendEth(send_account: string, to_address: string, send_token_amount: BigNumber) {
    const nonce = await ethers.provider.getTransactionCount(send_account, "latest");
    const gas_price = await ethers.provider.getGasPrice();
    
    const tx = {
      from: send_account,
      to: to_address,
      value: send_token_amount,
      nonce: nonce,
      gasLimit: 100000, // 100000
      gasPrice: gas_price,
    }
    const signer = ethers.provider.getSigner(send_account);
    await signer.sendTransaction(tx);

}

async function setSendRoot(cases: any, outbox: Contract) {
    const length = cases.length;
    for(let i = 0; i < length; i++) {
        await outbox.updateSendRoot(cases[i].root, cases[i].l2blockhash);
    }  
}

describe("Outbox", async function () {
    let outboxWithOpt: Contract;
    let outboxWithoutOpt: Contract;
    let bridge: Contract;
    const cases = TestCase.cases;
    const sentEthAmount = ethers.utils.parseEther("10");
    const rollupsAddress = "0xC12BA48c781F6e392B49Db2E25Cd0c28cD77531A";
    
    before(async function () {
        const accounts = await ethers.getSigners();
        const OutboxWithOpt = await ethers.getContractFactory("OutboxWithOptTester");
        const OutboxWithoutOpt = await ethers.getContractFactory("OutboxWithoutOptTester");
        const Bridge = await ethers.getContractFactory("BridgeTester");
        outboxWithOpt = await OutboxWithOpt.deploy();
        outboxWithoutOpt = await OutboxWithoutOpt.deploy();
        bridge = await Bridge.deploy();
        await bridge.initialize();
        await outboxWithOpt.initialize(rollupsAddress, bridge.address);
        await outboxWithoutOpt.initialize(rollupsAddress, bridge.address);

        await bridge.setOutbox(outboxWithOpt.address, true);
        await bridge.setOutbox(outboxWithoutOpt.address, true);

        await setSendRoot(cases, outboxWithOpt);
        await setSendRoot(cases, outboxWithoutOpt);

        await sendEth(accounts[0].address, bridge.address, sentEthAmount);
        
        
    })
    it("First call to initial some storage", async function () {    
        expect(await outboxWithOpt.executeTransaction(cases[0].proof, cases[0].index, cases[0].l2Sender, cases[0].to, cases[0].l2Block, cases[0].l1Block, cases[0].l2Timestamp, cases[0].value, cases[0].data)).to.emit(outboxWithOpt, "BridgeCallTriggered")
        expect(await outboxWithoutOpt.executeTransaction(cases[0].proof, cases[0].index, cases[0].l2Sender, cases[0].to, cases[0].l2Block, cases[0].l1Block, cases[0].l2Timestamp, cases[0].value, cases[0].data)).to.emit(outboxWithoutOpt, "BridgeCallTriggered")
        //await outboxWithOpt.executeTransaction(cases[0].proof,cases[0].index,cases[0].l2Sender,cases[0].to,cases[0].l2Block,cases[0].l1Block,cases[0].l2Timestamp,cases[0].value,cases[0].data);
    });
    
    it("Call twice without storage initail cost", async function () {
        expect(await outboxWithOpt.executeTransaction(cases[1].proof, cases[1].index, cases[1].l2Sender, cases[1].to, cases[1].l2Block, cases[1].l1Block, cases[1].l2Timestamp, cases[1].value, cases[1].data)).to.emit(outboxWithOpt, "BridgeCallTriggered")
        expect(await outboxWithoutOpt.executeTransaction(cases[1].proof, cases[1].index, cases[1].l2Sender, cases[1].to, cases[1].l2Block, cases[1].l1Block, cases[1].l2Timestamp, cases[1].value, cases[1].data)).to.emit(outboxWithoutOpt, "BridgeCallTriggered")
        
    });

    it("third call", async function () {
        expect(await outboxWithOpt.executeTransaction(cases[2].proof, cases[2].index, cases[2].l2Sender, cases[2].to, cases[2].l2Block, cases[2].l1Block, cases[2].l2Timestamp, cases[2].value, cases[2].data)).to.emit(outboxWithOpt, "BridgeCallTriggered")
        expect(await outboxWithoutOpt.executeTransaction(cases[2].proof, cases[2].index, cases[2].l2Sender, cases[2].to, cases[2].l2Block, cases[2].l1Block, cases[2].l2Timestamp, cases[2].value, cases[2].data)).to.emit(outboxWithoutOpt, "BridgeCallTriggered")
        
    });
    
  });