// SPDX-License-Identifier: UNLICENSED

pragma solidity ^0.8.0;

contract Contract {
  mapping(address => uint) public balances;

  event Transfer(address indexed from, address indexed to, uint value);

  constructor(uint256 initialSupply) {
    balances[msg.sender] = initialSupply;
  }

  function balanceOf(address account) public view returns (uint balance) {
    return balances[account];
  }

  function transfer(address to, uint amount) public returns (bool success) {
    return this.transferFrom(msg.sender, to, amount);
  }

  function transferFrom(
    address from,
    address to,
    uint amount
  ) public returns (bool success) {
    require(balances[from] >= amount);
    balances[from] -= amount;
    balances[to] += amount;
    emit Transfer(from, to, amount);
    return true;
  }

  function guessWhat(bool guess) public pure {
    assert(guess);
  }
}
