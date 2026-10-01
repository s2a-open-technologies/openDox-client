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
	export class RecordDetails {
	    id: number;
	    task_id: number;
	    summarize: string;
	
	    static createFrom(source: any = {}) {
	        return new RecordDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.task_id = source["task_id"];
	        this.summarize = source["summarize"];
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

