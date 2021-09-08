const { ethers, deployments } = require("hardhat");

describe("ValueArray", function () {
  it("Should pass ValueArrayTester", async function () {
    await deployments.fixture(["ValueArrayTester"]);

    const valueArrayTester = await ethers.getContract("ValueArrayTester");

    await valueArrayTester.test();
  });
});
