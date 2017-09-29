pragma solidity ^0.4.11;

import "./Owned.sol";
import "./StandardToken.sol";

contract ATMToken is StandardToken, Owned {
    // metadata
    string public constant name = "Attention Token of Media";
    string public constant symbol = "ATM";
    string public version = "1.0";
    uint256 public constant decimals = 8;
    bool public disabled = false;
    uint256 public constant MILLION = (10**6 * 10**decimals);
    // constructor
    function ATMToken(uint _amount,address testAddr) {
        totalSupply = 10000 * MILLION;
        balances[msg.sender] = _amount;
    }

    function getATMTotalSupply() external constant returns(uint256) {
        return totalSupply;
    }

    function setDisabled(bool flag) external onlyOwner {
        disabled = flag;
    }

    function transfer(address _to, uint256 _value) returns (bool success) {
        if(disabled){
            LogExecFalse("transfer is disabled.");
            return false;
        }
        return super.transfer(_to, _value);
    }

    function transferFrom(address _from, address _to, uint256 _value) returns (bool success) {
        if(disabled){
            LogExecFalse("transferFrom is disabled.");
            return false;
        }
        return super.transferFrom(_from, _to, _value);
    }
    function kill() external onlyOwner {
        selfdestruct(owner);
    }
}

