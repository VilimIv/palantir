export namespace main {
	
	export class CreateResult {
	    networkID: string;
	    virtualIP: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.networkID = source["networkID"];
	        this.virtualIP = source["virtualIP"];
	    }
	}
	export class  {
	    username: string;
	    virtualIP: string;
	
	    static createFrom(source: any = {}) {
	        return new (source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.virtualIP = source["virtualIP"];
	    }
	}
	export class JoinResult {
	    virtualIP: string;
	    peers: [];
	
	    static createFrom(source: any = {}) {
	        return new JoinResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.virtualIP = source["virtualIP"];
	        this.peers = this.convertValues(source["peers"], );
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
	export class PeerStatus {
	    username: string;
	    virtualIP: string;
	    mode: string;
	    ready: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PeerStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.virtualIP = source["virtualIP"];
	        this.mode = source["mode"];
	        this.ready = source["ready"];
	    }
	}

}

