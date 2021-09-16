const { ethers, run } = require("hardhat");
const fs = require("fs");
const assert = require("assert");

const PARALLEL = 128;

describe("OneStepProof", function () {
  const deployment = run("deploy", { "tags": "OneStepProofEntry" });
  const root = "./test/proofs/";
  const dir = fs.readdirSync(root);
  for (let file of dir) {
    if (!file.endsWith(".json")) continue;
    it("Should pass " + file + " proofs", async function () {
      await deployment;

      let path = root + file;
      let proofs = JSON.parse(fs.readFileSync(path));

      const osp = await ethers.getContract("OneStepProofEntry");

      const promises = [];
      for (let i = 0; i < proofs.length; i++) {
        process.stdout.write("\rTesting " + file + " proof " + i + "/" + proofs.length + " ");
        const proof = proofs[i];
        const promise = osp.proveOneStep([...Buffer.from(proof.before, "hex")], [...Buffer.from(proof.proof, "hex")])
          .catch(err => {
            console.error("Error executing proof " + i);
            throw err;
          })
          .then(after => assert.equal(after, "0x" + proof.after, "After state doesn't match after proof " + i));
        if (promises.length < PARALLEL) {
          promises.push(promise);
        } else {
          const finished = await Promise.race(promises.map((p, i) => p.then(_ => i)));
          promises[finished] = promise;
        }
      }
      await Promise.all(promises);
      process.stdout.write("\r");
    });
  }
});
