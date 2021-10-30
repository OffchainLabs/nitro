
pragma solidity >=0.4.21 <0.8.0;

interface ArbRentableStorage {
    function AllocateBin(uint binId) external;

    function GetBinTimeout(uint binId) external view returns(uint);

    function GetForeignBinTimeout(address binOwner, uint binId) external view returns(uint);

    function GetBinRenewGas(uint binId) external view returns(uint);

    function GetForeignBinRenewGas(address binOwner, uint binId)  external view returns(uint);

    function RenewBin(uint binId) external;

    function RenewForeignBin(address binOwner, uint binId) external;

    function SetInBin(uint binId, uint slot, bytes calldata value) external;

    function DeleteInBin(uint binId, uint slot) external;

    function GetInBin(uint binId, uint slot) external view returns(bytes memory);

    function GetInForeignBin(address binOwner, uint binId, uint slot) external view returns(bytes memory);
}
