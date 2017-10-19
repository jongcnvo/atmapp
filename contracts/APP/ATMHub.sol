pragma solidity ^0.4.11;
import "./ATMToken.sol";
import "./SafeMath.sol";
import "./Owned.sol";
contract ATMHub is SafeMath, Owned {
    
    uint public constant ATM = 100000000;
    uint8 public constant ROLE_ATMPLATFORM = 0;
    uint8 public constant ROLE_SCREEN = 1;
    uint8 public constant ROLE_OPERATOR = 2;
    uint8 public constant ROLE_USER = 3;
    
    struct publishStrategy {
        uint mediaID;
        uint mediaIndex;
        address advertiser;
        uint totalATMRewards;
        uint currentTotal;
        uint totalCount;
        uint currentCount;
        uint userPayPrice;   
        uint ratio1;    //rewards ratio
        uint ratio2; 
        uint ratio3; 
        bool enableIssueFlag;
    }
    ATMToken public atm;
    address public atmPlatformAddr;
    mapping(uint256 => publishStrategy) public mediaStrategy;
    mapping(address => uint256[]) public mediaList;
    uint256[] public freeMediaIndex;


    function ATMHub(address _ATMAddr){ 
        atm = ATMToken(_ATMAddr);  /*0x696a89e6dfa39b1c8ecf9ad86e826205ed17a8d8*/
        atmPlatformAddr = msg.sender;
    }

    /*
    create a new strategy structure
    */
    event LognewPublishStrategy(uint mediaID, string desc, 
        uint LogMediaIndex, address LogAdvertiser,
        uint LogTotalATMRewards,
        uint LogTotalCount, uint LogUserPayPrice,   
        uint[3] LogRatio);
    function newPublishStrategy(
        address _advertiser,
        uint _mediaID,
        uint _totalATMSupply,
        uint _totalCount,
        uint _userPayPrice,
        uint[3] _ratio) internal returns (publishStrategy strategy) {

        strategy.mediaID =  _mediaID;
        strategy.advertiser = _advertiser;
        strategy.totalATMRewards = _totalATMSupply;
        strategy.currentTotal = _totalATMSupply;
        strategy.totalCount = _totalCount;
        strategy.currentCount = _totalCount;
        strategy.userPayPrice = _userPayPrice;
        strategy.ratio1 = _ratio[0];
        strategy.ratio2 = _ratio[1];
        strategy.ratio3 = _ratio[2];
        strategy.mediaIndex = _mediaID;//addToMediaList(msg.sender, _mediaID);
        strategy.enableIssueFlag = true;
        LognewPublishStrategy(strategy.mediaID,"Publish Strategy ok.",
            strategy.mediaIndex,
            strategy.advertiser,strategy.totalATMRewards,
            strategy.totalCount,strategy.userPayPrice,   
            _ratio);
            
    }
    function getMediaStrategy(uint _mediaID) returns (address advertiser,
                                                        uint totalATMRewards,
                                                        uint currentTotal,
                                                        uint totalCount,
                                                        uint currentCount,
                                                        uint userPayPrice,   
                                                        uint ratio1,
                                                        uint ratio2, 
                                                        uint ratio3, 
                                                        bool enableIssueFlag){
                                                            
        publishStrategy strategy = mediaStrategy[_mediaID];
        totalATMRewards= strategy.totalATMRewards;  
        currentTotal   = strategy.currentTotal;     
        totalCount     = strategy.totalCount;  
        advertiser     = strategy.advertiser;  
        currentCount   = strategy.currentCount;     
        userPayPrice   = strategy.userPayPrice;     
        ratio1         = strategy.ratio1;           
        ratio2         = strategy.ratio2;           
        ratio3         = strategy.ratio3;           
        enableIssueFlag= strategy.enableIssueFlag; 
      return;
    }
    function withdrawATM(address _adversier, uint _amount) internal returns (bool ret){
        
        ret = atm.transferFrom(_adversier, _adversier, _amount);
    }

    event LogupdateAdvertiseFail(uint mediaID, string errinfo);
    function updateAdvertise(
        address _advertiser,   
        uint256 _mediaID, 
        uint256 _scanCount, 
        uint256 _price, 
        uint256[3] _atmRadio) onlyOwner { 
            
        if(mediaStrategy[_mediaID].mediaID==0){
            LogupdateAdvertiseFail(_mediaID,"mediaStrategy[_mediaID].mediaID==0");
            return ;
        }
        uint scanCount = _scanCount + mediaStrategy[_mediaID].currentCount;
        delete mediaStrategy[_mediaID];

        return newAdvertise(_advertiser, _mediaID, scanCount, _price, _atmRadio);   
    }
    
    function publishAdvertise(
        address _advertiser,   
        uint256 _mediaID, 
        uint256 _scanCount, 
        uint256 _price, 
        uint256[3] _atmRadio) onlyOwner  { 
        
        if(mediaStrategy[_mediaID].mediaID!=0){
            delete mediaStrategy[_mediaID];
        }
        newAdvertise(_advertiser,_mediaID,_scanCount,_price,_atmRadio);    
    }
    event LogPublishErr(uint mediaID, string errinfo, uint allowance, uint expect);
    function newAdvertise(
        address _advertiser,   
        uint256 _mediaID, 
        uint256 _scanCount, 
        uint256 _price, 
        uint256[3] _atmRadio) internal  { 

        uint totalATMSupply;

        totalATMSupply = _scanCount * _price / 100 * (100 + _atmRadio[ROLE_ATMPLATFORM]+_atmRadio[ROLE_SCREEN]+_atmRadio[ROLE_OPERATOR]);
        
        if(atm.allowance(_advertiser,this) < totalATMSupply ){
            LogPublishErr(_mediaID,"Advertier NOT have enough balance to publish advertise.",atm.allowance(_advertiser,this), totalATMSupply);
            return ;
        }
        if(_scanCount == 0 || _price == 0){
            LogPublishErr(_mediaID,"publish advertise: _scanCount == 0 || _price == 0 .", _scanCount, _price);
            return ;
        }
        mediaStrategy[_mediaID] = newPublishStrategy(
                                        _advertiser,
                                        _mediaID,
                                        totalATMSupply,
                                        _scanCount,
                                        _price,
                                        _atmRadio);
    }
    

    event LogdeleteAdvertise(uint mediaID, string info, uint balance, address to);
    event LogdeleteAdvertiseFail(uint mediaID, string err);
    event LogWithdrawATM (uint mediaID, string info, address advertiser, address atmPPool, uint amount);
    event LogWithdrawATMFail (uint mediaID, string info, address advertiser, address atmPPool, uint amount);
    function deleteAdvertise(uint256 _mediaID) onlyOwner {
    
       publishStrategy memory strategy = mediaStrategy[_mediaID];
       if(strategy.mediaID != _mediaID){
            LogdeleteAdvertiseFail(_mediaID,"delete advertise ok fail, strategy.mediaID != _mediaID.");
            return;
       }
       
       if(strategy.currentTotal > 0){
           if(withdrawATM(strategy.advertiser,strategy.currentTotal)==false){
                LogWithdrawATMFail(_mediaID,"withdrawATM fail.",this, strategy.advertiser, strategy.currentTotal);
                return;
            }else{
                LogWithdrawATM(_mediaID,"withdrawATM ok.",this, strategy.advertiser, strategy.currentTotal);
            }
       }

       delete mediaStrategy[_mediaID];
       LogdeleteAdvertise(_mediaID, "delete advertise ok.",strategy.currentTotal,strategy.advertiser);
    }
    /*
    calacute ATM rewards for everyone
    */
    function calcATMRewards(
        uint _price,
        uint[3] _ratio, 
        uint _scale) internal returns (uint[4] _amount){

        _amount[ROLE_ATMPLATFORM] = mul(div(_price, 100), _ratio[ROLE_ATMPLATFORM]) * _scale / 100; 
        _amount[ROLE_SCREEN] = mul(div(_price, 100), _ratio[ROLE_SCREEN]) * _scale / 100;  
        _amount[ROLE_OPERATOR] = mul(div(_price, 100), _ratio[ROLE_OPERATOR]) * _scale / 100;
        _amount[ROLE_USER] = _price * _scale / 100;
    }
    /*
    dispatch ATM
    */
    function transferATM(address _advertiser, address[4] _recipientList, uint[4] _amount) internal returns (bool ret){
        ret = true;
        for (uint i=0; i<4; i++){
            if( _amount[i] > 0 && _recipientList[i] != 0){
                ret = atm.transferFrom(_advertiser,_recipientList[i], _amount[i]);
                if(ret == false){
                    return;
                }
            }
        }
    }
    event LogpauseIssue(uint mediaID, string info);
    event LogpauseIssueErr(uint mediaID, string info);
    function pauseIssue(uint _mediaID) onlyOwner  {
        if(mediaStrategy[_mediaID].mediaID == 0){
            LogpauseIssueErr(_mediaID, "pauseIssue: mediaStrategy[_mediaID].mediaID == 0.");
            return ;
        }
        mediaStrategy[_mediaID].enableIssueFlag = false;
        LogpauseIssue(_mediaID, "pauseIssue ok.");
    }
    function enableIssue(uint _mediaID) onlyOwner  {
        if(mediaStrategy[_mediaID].mediaID == 0){
            LogpauseIssueErr(_mediaID, "enableIssue: mediaStrategy[_mediaID].mediaID == 0.");
            return ;
        }
        mediaStrategy[_mediaID].enableIssueFlag = true;
        LogpauseIssue(_mediaID,"enableIssue ok.");
    }
    
    event LogIssueReward(address user, string info, address[4] accounts, uint[4] rewards, uint[3] ratio);
    event LogIssueRewardParaErr(address user, string err, bool enable, uint  remainScanCount, address scaner, uint ID);
    event LogIssueRewardBalanceErr(address user, string err, uint ID, uint  remainBalance, uint expectBalance);
    event LogIssueRewardtransferATMErr(address user, string err, address ad, address[4] accounts, uint[4] rewards);
    function issueReward(address _userAddr, address _screenAddr, uint256 _adID, uint _scale) onlyOwner  {
        uint[4] memory rewards;
        address[4] memory accounts;
        uint rewardTotal;
        publishStrategy memory strategy = mediaStrategy[_adID];

        if (strategy.enableIssueFlag == false || 
            strategy.currentCount == 0 ||
            _userAddr == address(0) ||
            strategy.mediaID != _adID){
                
            LogIssueRewardParaErr(_userAddr, "issueReward para check error.",strategy.enableIssueFlag,strategy.currentCount,_userAddr,strategy.mediaID);
            return ;
        }

        accounts[ROLE_USER] = _userAddr;
        accounts[ROLE_ATMPLATFORM] = atmPlatformAddr;
        accounts[ROLE_SCREEN] = _screenAddr;
        accounts[ROLE_OPERATOR] = 0;
        
        uint[3] memory ratio = [strategy.ratio1,strategy.ratio2,strategy.ratio3];
        rewards = calcATMRewards(strategy.userPayPrice,ratio,_scale);
        /*rewardTotal = rewards[ROLE_ATMPLATFORM]+
                rewards[ROLE_SCREEN]+
                rewards[ROLE_OPERATOR]+
                rewards[ROLE_USER];*/
        rewardTotal = rewards[ROLE_USER];
        if(strategy.ratio1 != 0 && accounts[ROLE_ATMPLATFORM]!=0) rewardTotal += rewards[ROLE_ATMPLATFORM];
        if(strategy.ratio2 != 0&& accounts[ROLE_SCREEN]!=0) rewardTotal += rewards[ROLE_SCREEN];
        if(strategy.ratio3 != 0&& accounts[ROLE_OPERATOR]!=0) rewardTotal += rewards[ROLE_OPERATOR];
        if(mediaStrategy[_adID].currentTotal < rewardTotal){
            LogIssueRewardBalanceErr(_userAddr,"NOT enough balance to issue rewards.",_adID,mediaStrategy[_adID].currentTotal, rewardTotal);
            return ;
        }
        
        mediaStrategy[_adID].currentTotal -= rewardTotal;
        mediaStrategy[_adID].currentCount -= 1;
        if(transferATM(strategy.advertiser,accounts,rewards)==false){
            LogIssueRewardtransferATMErr(_userAddr,"when IssueReward, transferATM fail.",strategy.advertiser,accounts,rewards);
            return;
        }

        LogIssueReward(_userAddr,"user scan QR code sucess, Issue Reward ok!",accounts,rewards,ratio);
    }
}