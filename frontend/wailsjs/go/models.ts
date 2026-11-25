export namespace clientpolicy {
	
	export class Preferences {
	    theme?: string;
	    live_view?: boolean;
	    dictation_mic_mode?: string;
	    ui_language?: string;
	    dictation_language?: string;
	
	    static createFrom(source: any = {}) {
	        return new Preferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.live_view = source["live_view"];
	        this.dictation_mic_mode = source["dictation_mic_mode"];
	        this.ui_language = source["ui_language"];
	        this.dictation_language = source["dictation_language"];
	    }
	}

}

export namespace main {
	
	export class ActiveSession {
	    session_id: string;
	
	    static createFrom(source: any = {}) {
	        return new ActiveSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session_id = source["session_id"];
	    }
	}
	export class Category {
	    id: number;
	    name: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}
	export class InsertTarget {
	    process: string;
	    title: string;
	    list_active: boolean;
	    allowed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InsertTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.process = source["process"];
	        this.title = source["title"];
	        this.list_active = source["list_active"];
	        this.allowed = source["allowed"];
	    }
	}
	export class Preferences {
	    theme: string;
	    live_view: boolean;
	    dictation_mic_mode: string;
	    ui_language: string;
	    dictation_language: string;
	
	    static createFrom(source: any = {}) {
	        return new Preferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.live_view = source["live_view"];
	        this.dictation_mic_mode = source["dictation_mic_mode"];
	        this.ui_language = source["ui_language"];
	        this.dictation_language = source["dictation_language"];
	    }
	}
	export class Record {
	    session_id: string;
	    id: number;
	
	    static createFrom(source: any = {}) {
	        return new Record(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session_id = source["session_id"];
	        this.id = source["id"];
	    }
	}
	export class TranslationSegment {
	    missing_numbers: string[];
	    extra_numbers: string[];
	
	    static createFrom(source: any = {}) {
	        return new TranslationSegment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.missing_numbers = source["missing_numbers"];
	        this.extra_numbers = source["extra_numbers"];
	    }
	}
	export class RecordDetails {
	    id: number;
	    task_id: number;
	    summarize: string;
	    source_language?: string;
	    translation_language?: string;
	    translation_segments?: TranslationSegment[];
	
	    static createFrom(source: any = {}) {
	        return new RecordDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.task_id = source["task_id"];
	        this.summarize = source["summarize"];
	        this.source_language = source["source_language"];
	        this.translation_language = source["translation_language"];
	        this.translation_segments = this.convertValues(source["translation_segments"], TranslationSegment);
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
	export class Settings {
	    server_url: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server_url = source["server_url"];
	    }
	}
	export class TaskStatus {
	    task_id: number;
	    status: string;
	    result: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.task_id = source["task_id"];
	        this.status = source["status"];
	        this.result = source["result"];
	        this.error = source["error"];
	    }
	}
	export class TranslationAvailability {
	    available: boolean;
	    languages: string[];
	    target_language: string;
	    live_languages: string[];
	
	    static createFrom(source: any = {}) {
	        return new TranslationAvailability(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.languages = source["languages"];
	        this.target_language = source["target_language"];
	        this.live_languages = source["live_languages"];
	    }
	}
	
	export class UpdateInfo {
	    current: string;
	    latest: string;
	    available: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.available = source["available"];
	    }
	}
	export class WhisperSettings {
	    live_backend: string;
	
	    static createFrom(source: any = {}) {
	        return new WhisperSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.live_backend = source["live_backend"];
	    }
	}

}

