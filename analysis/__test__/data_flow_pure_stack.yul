{
    let a := 1
    let b := sload(a)
    let b2 := add(b, 2)
    let c := sload(b2)
    mstore(b, c)
}