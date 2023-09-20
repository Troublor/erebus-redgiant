{
  let a := sload(0)
  let b := sload(1)

  if eq(a, 0) {
    sstore(2, 0)
  }

  if eq(b, 1) {
    sstore(3, 1)
  }
}