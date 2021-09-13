const { ethers, run } = require("hardhat");

describe("ValueArray", function () {
  it("Should pass ValueArrayTester", async function () {
    await run("deploy", { tags: "ValueArrayTester" });

    const valueArrayTester = await ethers.getContract("ValueArrayTester");

    await valueArrayTester.test();
  });
});
