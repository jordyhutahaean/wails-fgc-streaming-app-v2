export namespace main {
	
	export class DoubleBracket {
	    players: string[];
	    scores: Record<string, Array<number>>;
	    winners: Record<string, string>;
	    losers: Record<string, string>;
	    meta: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new DoubleBracket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.players = source["players"];
	        this.scores = source["scores"];
	        this.winners = source["winners"];
	        this.losers = source["losers"];
	        this.meta = source["meta"];
	    }
	}
	export class SingleBracket {
	    players: string[];
	    scores: Record<string, Array<number>>;
	    winners: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new SingleBracket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.players = source["players"];
	        this.scores = source["scores"];
	        this.winners = source["winners"];
	    }
	}
	export class Bracket {
	    single: SingleBracket;
	    double: DoubleBracket;
	
	    static createFrom(source: any = {}) {
	        return new Bracket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.single = this.convertValues(source["single"], SingleBracket);
	        this.double = this.convertValues(source["double"], DoubleBracket);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Commentary {
	    commentator1: string;
	    description1: string;
	    commentator2: string;
	    description2: string;
	    visible: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Commentary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.commentator1 = source["commentator1"];
	        this.description1 = source["description1"];
	        this.commentator2 = source["commentator2"];
	        this.description2 = source["description2"];
	        this.visible = source["visible"];
	    }
	}
	
	export class Scoreboard {
	    game: string;
	    style: string;
	    titlecard: string;
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
	    score3: number;
	    visible3: boolean;
	    player4: string;
	    team4: string;
	    controller4: string;
	    score4: number;
	    visible4: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Scoreboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game = source["game"];
	        this.style = source["style"];
	        this.titlecard = source["titlecard"];
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
	        this.score3 = source["score3"];
	        this.visible3 = source["visible3"];
	        this.player4 = source["player4"];
	        this.team4 = source["team4"];
	        this.controller4 = source["controller4"];
	        this.score4 = source["score4"];
	        this.visible4 = source["visible4"];
	    }
	}

}

