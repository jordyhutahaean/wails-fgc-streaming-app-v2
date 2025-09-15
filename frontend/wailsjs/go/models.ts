export namespace main {
	
	export class Scoreboard {
	    game: string;
	    player1: string;
	    team1: string;
	    controller1: string;
	    score1: number;
	    visible1: boolean;
	    player2: string;
	    team2: string;
	    controller2: string;
	    score2: number;
	    visible2: boolean;
	    player3: string;
	    team3: string;
	    controller3: string;
	    visible3: boolean;
	    player4: string;
	    team4: string;
	    controller4: string;
	    visible4: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Scoreboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game = source["game"];
	        this.player1 = source["player1"];
	        this.team1 = source["team1"];
	        this.controller1 = source["controller1"];
	        this.score1 = source["score1"];
	        this.visible1 = source["visible1"];
	        this.player2 = source["player2"];
	        this.team2 = source["team2"];
	        this.controller2 = source["controller2"];
	        this.score2 = source["score2"];
	        this.visible2 = source["visible2"];
	        this.player3 = source["player3"];
	        this.team3 = source["team3"];
	        this.controller3 = source["controller3"];
	        this.visible3 = source["visible3"];
	        this.player4 = source["player4"];
	        this.team4 = source["team4"];
	        this.controller4 = source["controller4"];
	        this.visible4 = source["visible4"];
	    }
	}

}

