const { ethers, deployments } = require("hardhat");
const fs = require("fs");
const assert = require("assert");

describe("OneStepProof", function () {
  const root = "./test/proofs/";
  const dir = fs.readdirSync(root);
  for (let file of dir) {
    if (!file.endsWith(".json")) continue;
    it("Should pass " + file + " proofs", async function () {
      await deployments.fixture(["OneStepProofEntry"]);

      let path = root + file;
      let proofs = JSON.parse(fs.readFileSync(path));

      const osp = await ethers.getContract("OneStepProofEntry");

      for (let i = 0; i < proofs.length; i++) {
        const proof = proofs[i];
        const after = await osp.proveOneStep([...Buffer.from(proof.before, "hex")], [...Buffer.from(proof.proof, "hex")]);
        assert.equal(after, "0x" + proof.after, "After state doesn't match after proof " + i);
      }
    });
  }
});
