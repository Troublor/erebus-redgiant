pragma solidity ^0.8.0;

contract BasicStorage {
  uint a;
  uint[] b;
  mapping(uint => uint) c;

  function invoke(uint a_, uint b_, uint ck_, uint cv_) public {
    a = a_;
    b.push(b_);
    c[ck_] = cv_;
  }
}
