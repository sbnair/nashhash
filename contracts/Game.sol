pragma solidity ^0.4.23;



/*
!!!!!!!!!!! KNOWN BUGS !!!!!!!!!!!!!!!
1) 2/3 average consistent with the js results. BUT. The way we calculate it
in solidity seems to be rounded down at the taking of the average step, and then
again rounded down after division by 3. It is better to fix this because this skews
the average ever so slightly south. 
2) Protect against repeated reveal call. (SOLVED, needs testing)


!!!!!!!!!!!! POTENTIAL BUGS !!!!!!!!!!!!!!!!
1) Ether being brought over to the next round due to rounding issues. 
Check that this is not the case. 

!!!!!!!!!! LOGICAL IMPROVEMENTS !!!!!!!!!!!!
1) Send money to HOME address only once the fees reach a certain amount.
2) I have this idea of introducing a system of rounds so our players get a receit.
They can then use this receit to look up info about their game.

!!!!!!!!!! PERFORMANCE INMPROVEMENTS !!!!!!!!!!!!!!!!
1) Change the body of findWinners to use an in memory array to find the winners. 
Then copy those winners to info.lastWinners

!!!!!!!!!! REFACTORING GOALS !!!!!!!!!!!!!!!!!
1) Get a goddamn constructor in here with a reasonable amount of arguments
2) Maybe put all the state related variable into a State struct? Work around that object. 
I started the idea....
3) Give normal names to everything. Also, change to cammelcase. 
Apparently, the lower_case_underscore is not that popular in Solidity...


*/


/*
Dear all, games should oinherit form this contract because this contract has the commit/reveal protocol
The specific game will have to only define two functions:

-- guessCheck(string guess): the function has to error out in if the guess is not compliant with the rules
-- findWinners(): the function determined the winners and distributes the payouts. 

*/

import "./Pausable.sol";
import "./GameHelper.sol";


contract Game is Pausable, GameHelper {

    uint public birthBlock = block.number;
    
    enum GameState {COMMIT_STATE, REVEAL_STATE, PAYOUT_STATE}

    // This is the idea.
    struct State {
        GameState gameState;
        uint gameStateDebug;
        uint currNumberCommits;
        uint currNumberReveals;
        uint finalCommitBlock;
        uint round;
    }

    struct Config {
        uint REVEAL_PERIOD;
        uint MAX_PLAYERS;

        address FEE_ADDRESS;
        uint GAME_FEE_PERCENT;
        uint STAKE_SIZE;
    }

    struct GameInfo {
        address[] lastWinners;
        uint lastPrize;
    }

    Config internal config;
    State internal state;
    GameInfo internal info;

    constructor(uint maxp) public {
        owner = msg.sender;

        config.REVEAL_PERIOD = 5;
        config.GAME_FEE_PERCENT = 5;
        config.MAX_PLAYERS = maxp;
        config.STAKE_SIZE = 1 ether;
        config.FEE_ADDRESS = 0x2540099e9ed04aF369d557a40da2D8f9c2ab928D;

        state.gameState = GameState.COMMIT_STATE;
        state.currNumberCommits = 0;
        state.currNumberReveals = 0;
        state.finalCommitBlock = 0;

        info.lastPrize = 0;
    }

    // Contrcact public API
    function getGameState() public view returns(uint) {
        return state.gameStateDebug;
    }

    function getCurrentCommits() public view returns(uint) {
        return state.currNumberCommits;
    }

    function getCurrentReveals() public view returns(uint) {
        return state.currNumberReveals;
    }

    function getStakeSize() public view returns(uint) {
        return config.STAKE_SIZE;
    }

    function getNumberOfWinners() public view returns(uint) {
        return info.lastWinners.length;
    }

    function getLastWinners(uint i) public view returns(address){
        return info.lastWinners[i];
    }

    function getLastPrize() public view returns(uint){
        return info.lastPrize;
    }

    function getGameFee() public view returns(uint){
        return config.GAME_FEE_PERCENT;
    }

    function getMaxPlayers() public view returns(uint){
        return config.MAX_PLAYERS;
    }

    function setMaxPlayers(uint new_max) public onlyOwner {
        config.MAX_PLAYERS = new_max;
    }
    


    // Commit/Reveal Protocol vars
    mapping (address => bytes32) public commits;
    address[] internal commitsKeys;


    mapping (address => string) public gameData;
    address[] internal gameDataKeys;

    ////// DEBUG vars and debug functions
    uint public average23 = 0;

    // Reset the contract to the initial state
    function reset() public onlyOwner {  
        toCommitState();
        info.lastPrize = 0;
        delete info.lastWinners;
    }

    event CommitsSubmitted(
        uint round
    );

    function commit(bytes32 hashedCommit) public payable whenNotPaused {
        
        require(state.gameState == GameState.COMMIT_STATE);

        require(msg.value == config.STAKE_SIZE);

        commits[msg.sender] = hashedCommit;
        commitsKeys.push(msg.sender);
        state.currNumberCommits++;

        // If we received the MAX_PLAYER number of commits, it is time for
        // us to change state.
        if (state.currNumberCommits == config.MAX_PLAYERS) {
            toRevealState();
            emit CommitsSubmitted(1);
        }
    }

    event RevealsSubmitted();

    event DebugEvent(
        string error
    );
    function reveal(string guess, string random) public whenNotPaused {
        

        require(state.gameState == GameState.REVEAL_STATE);
        
        // DEBUG: Need to make sure it throws if the guess is not integer
        //uint guess_num = stringToUint(guess);
        
        checkGuess(guess);

        // Check that the hashes match
        require(commits[msg.sender] == keccak256(guess, random));

        // When they do, we add the revealed guess to game data
        gameData[msg.sender] = guess;
        gameDataKeys.push(msg.sender);
        state.currNumberReveals++;

        if(state.currNumberReveals == config.MAX_PLAYERS){
            emit RevealsSubmitted();
            state.gameState = GameState.PAYOUT_STATE;
            state.gameStateDebug = 2;
        }
    }

    function checkGuess(string guess) private;

    function payout() public onlyOwner whenNotPaused {
        require(state.gameState == GameState.PAYOUT_STATE);
        findWinners();
        toCommitState();
    }



    event DebugWinner(address addr, uint n);

    function findWinners() private;

    // Call this funtion to get to COMMIT_STATE
    event DebugCommitState(uint last_win_l, uint win_l);

    function toCommitState() internal {

        state.gameState = GameState.COMMIT_STATE;
        state.gameStateDebug = 0;

        uint i;
        for(i = 0; i < commitsKeys.length; i++){
            delete commits[commitsKeys[i]];
        }
        delete commitsKeys;

        delete gameDataKeys;
        
        state.currNumberCommits = 0;
        state.currNumberReveals = 0;

    }

    // Call this function to get to REVEAL_STATE
    function toRevealState() internal {

        state.gameState = GameState.REVEAL_STATE;
        state.gameStateDebug = 1;
        state.finalCommitBlock = block.number;

    }

}