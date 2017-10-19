const ATMHub= artifacts.require("ATMHub");
const ATMToken = artifacts.require("ATMToken");

contract('发布广告/扫码/删除广告 : ', function(accounts) {
  let atm;
  let atmhub;
  var ATM = 100000000;
  var paltform  = accounts[0];
  var advertiser = accounts[1];
  var screen = accounts[2];
  var user = accounts[3];

  it("Case1: 1个广告,限制5次,用户扫码6次,删除广告", async function() {

    atm = await ATMToken.new(100000*ATM);
    atmhub = await ATMHub.new(atm.address);

    console.log('ATMTotalSupply: '+(await atm.getATMTotalSupply()));
  
    await atm.transfer(advertiser,100000*ATM);
    assert.equal((await atm.balanceOf(advertiser)).toNumber(), 100000*ATM);

    await atm.approve(atmhub.address,9*ATM,{from: advertiser});
    assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 9*ATM);

    /*发布广告: id=12121 扫码次数:5 每次给用户:1ATM 其他分成比例: [50,30,20]*/ 
    var scanCount = 5;
    var mediaID1 = 12121;
    await atmhub.publishAdvertise(advertiser,mediaID1,scanCount,1*ATM,[50,30,0]);
    console.log('mediaStrategy: '+(await atmhub.mediaStrategy(mediaID1)));
    /* mediaStrategy的数据结构
        uint mediaID; 0
        uint mediaIndex; 1
        address advertiser; 2
        uint totalATMRewards; 3
        uint currentTotal; 4
        uint totalCount; 5
        uint currentCount; 6
        uint userPayPrice;    7
        uint ratio1;    //rewards ratio 8
        uint ratio2; 
        uint ratio3; 
        bool enableIssueFlag; 9
    */
    assert.equal((await atmhub.mediaStrategy(mediaID1))[0],mediaID1);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[2],advertiser);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[3].toNumber(),1*ATM*1.8*scanCount);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[6],5);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[8],50);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[9],30);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[10],0);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[11],true);
    /* 1次扫码 */
    await atmhub.issueReward(user,screen,mediaID1,100);
    assert.equal((await atm.balanceOf(user)).toNumber(), 1*ATM);
    assert.equal((await atm.balanceOf(paltform)).toNumber(), 0.5*ATM);
    assert.equal((await atm.balanceOf(screen)).toNumber(), 0.3*ATM);
    /* 2次扫码 */
    await atmhub.issueReward(user,screen,mediaID1,100);
    assert.equal((await atm.balanceOf(user)).toNumber(), 2*ATM);
    assert.equal((await atm.balanceOf(paltform)).toNumber(), 1*ATM);
    assert.equal((await atm.balanceOf(screen)).toNumber(), 0.6*ATM);
    /* 3次扫码 */
    await atmhub.issueReward(user,screen,mediaID1,100);
    assert.equal((await atm.balanceOf(user)).toNumber(), 3*ATM);
    assert.equal((await atm.balanceOf(paltform)).toNumber(), 1.5*ATM);
    assert.equal((await atm.balanceOf(screen)).toNumber(), 0.9*ATM);
    /* 4次扫码 */
    await atmhub.issueReward(user,screen,mediaID1,100);
    assert.equal((await atm.balanceOf(user)).toNumber(), 4*ATM);
    assert.equal((await atm.balanceOf(paltform)).toNumber(), 2.0*ATM);
    assert.equal((await atm.balanceOf(screen)).toNumber(), 1.2*ATM);
    /* 5次扫码 */
    await atmhub.issueReward(user,screen,mediaID1,100);
    assert.equal((await atm.balanceOf(user)).toNumber(), 5*ATM);
    assert.equal((await atm.balanceOf(paltform)).toNumber(), 2.5*ATM);
    assert.equal((await atm.balanceOf(screen)).toNumber(), 1.5*ATM);
    /* 6次扫码 */
    await atmhub.issueReward(user,screen,mediaID1,100);
    assert.equal((await atm.balanceOf(user)).toNumber(), 5*ATM);
    assert.equal((await atm.balanceOf(paltform)).toNumber(), 2.5*ATM);
    assert.equal((await atm.balanceOf(screen)).toNumber(), 1.5*ATM);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[6],0);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[4].toNumber(),0*ATM);
    assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 0*ATM);
    assert.equal((await atm.balanceOf(advertiser)).toNumber(), 99991*ATM);

    /*删除广告 */
    await atmhub.deleteAdvertise(mediaID1);
    assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 0);
    assert.equal((await atm.balanceOf(advertiser)).toNumber(), 99991*ATM);
  });

  it("Case2: 1个广告,限制5次,用户扫码2次,删除广告", async function() {
    
    atm = await ATMToken.new(100000*ATM);
    atmhub = await ATMHub.new(atm.address);

    console.log('ATMTotalSupply: '+(await atm.getATMTotalSupply()));
    
    await atm.transfer(advertiser,100000*ATM);
    assert.equal((await atm.balanceOf(advertiser)).toNumber(), 100000*ATM);

    await atm.approve(atmhub.address,10*ATM,{from: advertiser});
    assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 10*ATM);

    /*发布广告: id=12121 扫码次数:5 每次给用户:1ATM 其他分成比例: [50,30,20]*/ 
    var scanCount = 5;
    var mediaID1 = 12121;
    await atmhub.publishAdvertise(advertiser,mediaID1,scanCount,1*ATM,[50,30,20]);
    console.log('mediaStrategy: '+(await atmhub.mediaStrategy(mediaID1)));
    /* mediaStrategy的数据结构
        uint mediaID; 0
        uint mediaIndex; 1
        address advertiser; 2
        uint totalATMRewards; 3
        uint currentTotal; 4
        uint totalCount; 5
        uint currentCount; 6
        uint userPayPrice;    7
        uint ratio1;    //rewards ratio 8
        uint ratio2; 
        uint ratio3; 
        bool enableIssueFlag; 9
    */
    assert.equal((await atmhub.mediaStrategy(mediaID1))[0],mediaID1);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[2],advertiser);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[3].toNumber(),1*ATM*2*scanCount);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[6],5);
    /* 1次扫码 */
    await atmhub.issueReward(user,screen,mediaID1,100);
    assert.equal((await atm.balanceOf(user)).toNumber(), 1*ATM);
    assert.equal((await atm.balanceOf(paltform)).toNumber(), 0.5*ATM);
    assert.equal((await atm.balanceOf(screen)).toNumber(), 0.3*ATM);
    /* 2次扫码 */
    await atmhub.issueReward(user,screen,mediaID1,100);
    assert.equal((await atm.balanceOf(user)).toNumber(), 2*ATM);
    assert.equal((await atm.balanceOf(paltform)).toNumber(), 1*ATM);
    assert.equal((await atm.balanceOf(screen)).toNumber(), 0.6*ATM);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[6],3);
    assert.equal((await atmhub.mediaStrategy(mediaID1))[4].toNumber(),6.4*ATM);
    assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 6.4*ATM);
    assert.equal((await atm.balanceOf(advertiser)).toNumber(), 99996.4*ATM);

    /*删除广告 */
    await atmhub.deleteAdvertise(mediaID1);
    assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 0);
    assert.equal((await atm.balanceOf(advertiser)).toNumber(), 99996.4*ATM);
    });

    it("Case3: 1个广告,限制5次,删除广告后用户扫码1次,", async function() {
    
        atm = await ATMToken.new(100000*ATM);
        atmhub = await ATMHub.new(atm.address);

        console.log('ATMTotalSupply: '+(await atm.getATMTotalSupply()));
        
        await atm.transfer(advertiser,100000*ATM);
        assert.equal((await atm.balanceOf(advertiser)).toNumber(), 100000*ATM);

        await atm.approve(atmhub.address,10*ATM,{from: advertiser});
        assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 10*ATM);

        /*发布广告: id=12121 扫码次数:5 每次给用户:1ATM 其他分成比例: [50,30,20]*/ 
        var scanCount = 5;
        var mediaID1 = 12121;
        await atmhub.publishAdvertise(advertiser,mediaID1,scanCount,1*ATM,[50,30,20]);
        console.log('mediaStrategy: '+(await atmhub.mediaStrategy(mediaID1)));
        /* mediaStrategy的数据结构
        uint mediaID; 0
        uint mediaIndex; 1
        address advertiser; 2
        uint totalATMRewards; 3
        uint currentTotal; 4
        uint totalCount; 5
        uint currentCount; 6
        uint userPayPrice;    7
        uint ratio1;    //rewards ratio 8
        uint ratio2; 
        uint ratio3; 
        bool enableIssueFlag; 9
        */
        assert.equal((await atmhub.mediaStrategy(mediaID1))[0],mediaID1);
        assert.equal((await atmhub.mediaStrategy(mediaID1))[2],advertiser);
        assert.equal((await atmhub.mediaStrategy(mediaID1))[3].toNumber(),1*ATM*2*scanCount);
        assert.equal((await atmhub.mediaStrategy(mediaID1))[6],5);

        /*删除广告 */
        await atmhub.deleteAdvertise(mediaID1);
        assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 0);
        
        assert.equal((await atm.balanceOf(advertiser)).toNumber(), 100000*ATM);

        /* 1次扫码 */
        await atmhub.issueReward(user,screen,mediaID1,100);
        assert.equal((await atm.balanceOf(user)).toNumber(), 0);
        assert.equal((await atm.balanceOf(paltform)).toNumber(), 0);
        assert.equal((await atm.balanceOf(screen)).toNumber(), 0);
        assert.equal((await atmhub.mediaStrategy(mediaID1))[6],0);
        assert.equal((await atmhub.mediaStrategy(mediaID1))[4].toNumber(),0);
        assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 0);
        assert.equal((await atm.balanceOf(advertiser)).toNumber(), 100000*ATM);
    });

    it("Case4: 1个广告,限制5次,用户注意力度量50%扫码1次,用户注意力度量10%扫码1次", async function() {
        
        atm = await ATMToken.new(100000*ATM);
        atmhub = await ATMHub.new(atm.address);

        console.log('ATMTotalSupply: '+(await atm.getATMTotalSupply()));
        
        await atm.transfer(advertiser,100000*ATM);
        assert.equal((await atm.balanceOf(advertiser)).toNumber(), 100000*ATM);

        await atm.approve(atmhub.address,10*ATM,{from: advertiser});
        assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 10*ATM);

        /*发布广告: id=12121 扫码次数:5 每次给用户:1ATM 其他分成比例: [50,30,20]*/ 
        var scanCount = 5;
        var mediaID1 = 12121;
        await atmhub.publishAdvertise(advertiser,mediaID1,scanCount,1*ATM,[50,30,20]);
        console.log('mediaStrategy: '+(await atmhub.mediaStrategy(mediaID1)));
        /* mediaStrategy的数据结构
        uint mediaID; 0
        uint mediaIndex; 1
        address advertiser; 2
        uint totalATMRewards; 3
        uint currentTotal; 4
        uint totalCount; 5
        uint currentCount; 6
        uint userPayPrice;    7
        uint ratio1;    //rewards ratio 8
        uint ratio2; 
        uint ratio3; 
        bool enableIssueFlag; 9
        */
        assert.equal((await atmhub.mediaStrategy(mediaID1))[0],mediaID1);
        assert.equal((await atmhub.mediaStrategy(mediaID1))[2],advertiser);
        assert.equal((await atmhub.mediaStrategy(mediaID1))[3].toNumber(),1*ATM*2*scanCount);
        assert.equal((await atmhub.mediaStrategy(mediaID1))[6],5);

        /* 1次扫码 */
        await atmhub.issueReward(user,screen,mediaID1,50);
        assert.equal((await atm.balanceOf(user)).toNumber(), 0.5*ATM);
        assert.equal((await atm.balanceOf(paltform)).toNumber(), 0.25*ATM);
        assert.equal((await atm.balanceOf(screen)).toNumber(), 0.15*ATM);
        /* 2次扫码 */
        await atmhub.issueReward(user,screen,mediaID1,10);
        assert.equal((await atm.balanceOf(user)).toNumber(), 0.6*ATM);
        assert.equal((await atm.balanceOf(paltform)).toNumber(), 0.3*ATM);
        assert.equal((await atm.balanceOf(screen)).toNumber(), 0.18*ATM);
        
        assert.equal((await atmhub.mediaStrategy(mediaID1))[6],3);
        assert.equal((await atmhub.mediaStrategy(mediaID1))[4].toNumber(),8.92*ATM);
        assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 8.92*ATM);
        assert.equal((await atm.balanceOf(advertiser)).toNumber(), 99998.92*ATM);

        /*删除广告 */
        await atmhub.deleteAdvertise(mediaID1);
        assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 0);
        assert.equal((await atm.balanceOf(advertiser)).toNumber(), 99998.92*ATM);
    });
    it("Case5: 3个广告,限制5次,用户分别扫码1次", async function() {
        
        atm = await ATMToken.new(100000*ATM);
        atmhub = await ATMHub.new(atm.address);

        console.log('ATMTotalSupply: '+(await atm.getATMTotalSupply()));
        
        await atm.transfer(advertiser,100000*ATM);
        assert.equal((await atm.balanceOf(advertiser)).toNumber(), 100000*ATM);

        await atm.approve(atmhub.address,30*ATM,{from: advertiser});
        assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 30*ATM);

        /*发布广告: id=12121.12122,12123 扫码次数:5 每次给用户:1ATM 其他分成比例: [50,30,20]*/ 
        var scanCount = 5;
        var mediaID1 = 12121;
        var mediaID2 = 12122;
        var mediaID3 = 12123;
        await atmhub.publishAdvertise(advertiser,mediaID1,scanCount,1*ATM,[50,30,20]);
        console.log('mediaStrategy: '+(await atmhub.mediaStrategy(mediaID1)));
        await atmhub.publishAdvertise(advertiser,mediaID2,scanCount,1*ATM,[50,30,20]);
        console.log('mediaStrategy: '+(await atmhub.mediaStrategy(mediaID2)));
        await atmhub.publishAdvertise(advertiser,mediaID3,scanCount,1*ATM,[50,30,20]);
        console.log('mediaStrategy: '+(await atmhub.mediaStrategy(mediaID3)));

        /* 1次扫码 */
        await atmhub.issueReward(user,screen,mediaID1,50);
        await atmhub.issueReward(user,screen,mediaID2,50);
        await atmhub.issueReward(user,screen,mediaID3,50);
        assert.equal((await atm.balanceOf(user)).toNumber(), 1.5*ATM);
        assert.equal((await atm.balanceOf(paltform)).toNumber(), 0.75*ATM);
        assert.equal((await atm.balanceOf(screen)).toNumber(), 0.45*ATM);

    });

    it("Case6: 1个广告,限制5次,用户扫码1次,追加扫码5次", async function() {
        
        atm = await ATMToken.new(100000*ATM);
        atmhub = await ATMHub.new(atm.address);

        console.log('ATMTotalSupply: '+(await atm.getATMTotalSupply()));
        
        await atm.transfer(advertiser,100000*ATM);
        assert.equal((await atm.balanceOf(advertiser)).toNumber(), 100000*ATM);

        await atm.approve(atmhub.address,20*ATM,{from: advertiser});
        assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 20*ATM);

        /*发布广告: id=12121.12122,12123 扫码次数:5 每次给用户:1ATM 其他分成比例: [50,30,20]*/ 
        var scanCount = 5;
        var mediaID1 = 12121;
        await atmhub.publishAdvertise(advertiser,mediaID1,scanCount,1*ATM,[50,30,20]);
        console.log('mediaStrategy: '+(await atmhub.mediaStrategy(mediaID1)));

        /* 1次扫码 */
        await atmhub.issueReward(user,screen,mediaID1,50);

        /**追加扫码5次 */
        await atmhub.updateAdvertise(advertiser,mediaID1,scanCount,1*ATM,[50,30,20]);
        console.log('mediaStrategy: '+(await atmhub.mediaStrategy(mediaID1)));

        assert.equal((await atmhub.mediaStrategy(mediaID1))[6],9);
    });

    it("Case7: 1个广告,限制5次,去使能分成,用户扫码无效,再使能,扫码有效", async function() {
        
        atm = await ATMToken.new(100000*ATM);
        atmhub = await ATMHub.new(atm.address);

        console.log('ATMTotalSupply: '+(await atm.getATMTotalSupply()));
        
        await atm.transfer(advertiser,100000*ATM);
        assert.equal((await atm.balanceOf(advertiser)).toNumber(), 100000*ATM);

        await atm.approve(atmhub.address,10*ATM,{from: advertiser});
        assert.equal((await atm.allowance(advertiser,atmhub.address)).toNumber(), 10*ATM);

        /*发布广告: id=12121.12122,12123 扫码次数:5 每次给用户:1ATM 其他分成比例: [50,30,20]*/ 
        var scanCount = 5;
        var mediaID1 = 12121;
        await atmhub.publishAdvertise(advertiser,mediaID1,scanCount,1*ATM,[50,30,20]);
        console.log('mediaStrategy: '+(await atmhub.mediaStrategy(mediaID1)));

        await atmhub.pauseIssue(mediaID1);
        
        /* 1次扫码 */
        await atmhub.issueReward(user,screen,mediaID1,100);
        assert.equal((await atm.balanceOf(user)).toNumber(), 0);
        assert.equal((await atm.balanceOf(paltform)).toNumber(), 0);
        assert.equal((await atm.balanceOf(screen)).toNumber(), 0);
        
        await atmhub.enableIssue(mediaID1);
        
        /* 1次扫码 */
        await atmhub.issueReward(user,screen,mediaID1,100);
        assert.equal((await atm.balanceOf(user)).toNumber(), 1*ATM);
        assert.equal((await atm.balanceOf(paltform)).toNumber(), 0.5*ATM);
        assert.equal((await atm.balanceOf(screen)).toNumber(), 0.3*ATM);

    });
}); 
  