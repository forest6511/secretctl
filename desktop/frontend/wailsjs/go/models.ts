export namespace main {
	
	export class AuditLogEntry {
	    timestamp: string;
	    action: string;
	    source: string;
	    key?: string;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.action = source["action"];
	        this.source = source["source"];
	        this.key = source["key"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class AuditLogFilter {
	    action?: string;
	    source?: string;
	    key?: string;
	    startTime?: string;
	    endTime?: string;
	    success?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AuditLogFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.source = source["source"];
	        this.key = source["key"];
	        this.startTime = source["startTime"];
	        this.endTime = source["endTime"];
	        this.success = source["success"];
	    }
	}
	export class AuthStatus {
	    unlocked: boolean;
	    vaultDir: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unlocked = source["unlocked"];
	        this.vaultDir = source["vaultDir"];
	    }
	}
	export class Secret {
	    key: string;
	    value?: string;
	    notes?: string;
	    url?: string;
	    tags?: string[];
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Secret(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.notes = source["notes"];
	        this.url = source["url"];
	        this.tags = source["tags"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class SecretListItem {
	    key: string;
	    tags?: string[];
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SecretListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.tags = source["tags"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}

