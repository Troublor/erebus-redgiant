// SPDX-License-Identifier: UNLICENSED

pragma solidity ^0.8.0;

contract Contract {
  address[] public array;
  mapping(address => uint) public map;
  uint public value;

  struct T {
    address[] array;
    mapping(address => uint[]) map;
    uint value;
  }
  mapping(address => T) public t_map;

  function setValue(uint _value) public {
    value = _value;
  }

  function addAddress(address _address) public {
    array.push(_address);
  }

  function setAddressValue(address _address, uint _value) public {
    map[_address] = _value;
  }

  function setT(address _addr, uint _value) public {
    T storage t = t_map[_addr];
    t.array.push(_addr);
    t.map[_addr].push(_value);
  }
}
