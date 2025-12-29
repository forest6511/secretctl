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
	export class AuditLogSearchResult {
	    entries: AuditLogEntry[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new AuditLogSearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entries = this.convertValues(source["entries"], AuditLogEntry);
	        this.total = source["total"];
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
	export class FieldDTO {
	    value: string;
	    sensitive: boolean;
	    aliases?: string[];
	    kind?: string;
	    hint?: string;
	
	    static createFrom(source: any = {}) {
	        return new FieldDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.sensitive = source["sensitive"];
	        this.aliases = source["aliases"];
	        this.kind = source["kind"];
	        this.hint = source["hint"];
	    }
	}
	export class Secret {
	    key: string;
	    value?: string;
	    fields?: Record<string, FieldDTO>;
	    fieldOrder?: string[];
	    bindings?: Record<string, string>;
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
	        this.fields = this.convertValues(source["fields"], FieldDTO, true);
	        this.fieldOrder = source["fieldOrder"];
	        this.bindings = source["bindings"];
	        this.notes = source["notes"];
	        this.url = source["url"];
	        this.tags = source["tags"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class SecretListItem {
	    key: string;
	    tags?: string[];
	    updatedAt: string;
	    fieldCount: number;
	    bindingCount: number;
	    hasNotes: boolean;
	    hasUrl: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SecretListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.tags = source["tags"];
	        this.updatedAt = source["updatedAt"];
	        this.fieldCount = source["fieldCount"];
	        this.bindingCount = source["bindingCount"];
	        this.hasNotes = source["hasNotes"];
	        this.hasUrl = source["hasUrl"];
	    }
	}
	export class SecretUpdateDTO {
	    key: string;
	    fields: Record<string, FieldDTO>;
	    bindings?: Record<string, string>;
	    notes?: string;
	    url?: string;
	    tags?: string[];
	
	    static createFrom(source: any = {}) {
	        return new SecretUpdateDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.fields = this.convertValues(source["fields"], FieldDTO, true);
	        this.bindings = source["bindings"];
	        this.notes = source["notes"];
	        this.url = source["url"];
	        this.tags = source["tags"];
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
	export class TemplateFieldInfo {
	    name: string;
	    sensitive: boolean;
	    hint: string;
	
	    static createFrom(source: any = {}) {
	        return new TemplateFieldInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.sensitive = source["sensitive"];
	        this.hint = source["hint"];
	    }
	}
	export class TemplateInfo {
	    id: string;
	    name: string;
	    description: string;
	    icon: string;
	    fields: TemplateFieldInfo[];
	    bindings: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new TemplateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.icon = source["icon"];
	        this.fields = this.convertValues(source["fields"], TemplateFieldInfo);
	        this.bindings = source["bindings"];
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

