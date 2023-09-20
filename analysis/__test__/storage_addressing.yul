{
    let seed := 9
    mstore(0, caller())
    mstore(0x20, seed)
    let addr := keccak256(0, 0x40)
    let sdata := sload(addr)
    sstore(1, sdata)
}