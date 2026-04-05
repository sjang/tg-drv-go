export namespace storage {
	
	export class File {
	    id: string;
	    folder_id: string;
	    name: string;
	    size: number;
	    mime_type: string;
	    sha256_hash: string;
	    message_id: number;
	    telegram_file_id?: string;
	    has_thumbnail: boolean;
	    // Go type: time
	    upload_date: any;
	    is_duplicate: boolean;
	    source_file_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new File(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.folder_id = source["folder_id"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.mime_type = source["mime_type"];
	        this.sha256_hash = source["sha256_hash"];
	        this.message_id = source["message_id"];
	        this.telegram_file_id = source["telegram_file_id"];
	        this.has_thumbnail = source["has_thumbnail"];
	        this.upload_date = this.convertValues(source["upload_date"], null);
	        this.is_duplicate = source["is_duplicate"];
	        this.source_file_id = source["source_file_id"];
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
	export class Folder {
	    id: string;
	    name: string;
	    channel_id: number;
	    access_hash: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    file_count?: number;
	    total_size?: number;
	    hidden: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Folder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.channel_id = source["channel_id"];
	        this.access_hash = source["access_hash"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.file_count = source["file_count"];
	        this.total_size = source["total_size"];
	        this.hidden = source["hidden"];
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

export namespace telegram {
	
	export class AuthStatus {
	    authenticated: boolean;
	    phone_number?: string;
	    first_name?: string;
	    last_name?: string;
	    username?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authenticated = source["authenticated"];
	        this.phone_number = source["phone_number"];
	        this.first_name = source["first_name"];
	        this.last_name = source["last_name"];
	        this.username = source["username"];
	    }
	}

}

