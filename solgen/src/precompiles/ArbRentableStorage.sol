
pragma solidity >=0.4.21 <0.8.0;

interface ArbRentableStorage {
    function AllocateBin(uint id) external;

    function GetBinTimeout(uint id) external view returns(uint);

    function GetBinRenewGas(uint id) external view returns(uint);

    function SetInBin(uint id, uint slot, bytes calldata value) external;

    function DeleteInBin(uint id, uint slot) external;

    function GetInBin(uint id, uint slot) external view returns(bytes memory);
}
