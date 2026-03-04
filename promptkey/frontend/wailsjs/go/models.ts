export namespace main {
	
	export class Provider {
	    name: string;
	    baseURL: string;
	    apiKey: string;
	    model: string;
	    systemPrompt: string;
	
	    static createFrom(source: any = {}) {
	        return new Provider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.baseURL = source["baseURL"];
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.systemPrompt = source["systemPrompt"];
	    }
	}
	export class Config {
	    hotkey: string;
	    providers: Provider[];
	    activeProvider: string;
	    resultWidth: number;
	    resultHeight: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hotkey = source["hotkey"];
	        this.providers = this.convertValues(source["providers"], Provider);
	        this.activeProvider = source["activeProvider"];
	        this.resultWidth = source["resultWidth"];
	        this.resultHeight = source["resultHeight"];
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

}

