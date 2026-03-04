export namespace main {
	
	export class ContextConfig {
	    initialized: boolean;
	    enabled: boolean;
	    clipboard: boolean;
	    activeApp: boolean;
	    dateTime: boolean;
	    osLocale: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContextConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.initialized = source["initialized"];
	        this.enabled = source["enabled"];
	        this.clipboard = source["clipboard"];
	        this.activeApp = source["activeApp"];
	        this.dateTime = source["dateTime"];
	        this.osLocale = source["osLocale"];
	    }
	}
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
	    context: ContextConfig;
	
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
	        this.context = this.convertValues(source["context"], ContextConfig);
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

