export namespace config {
	
	export class Snapshot {
	    config_path: string;
	    repo_root: string;
	    remote_url: string;
	    vault_file_name: string;
	    load_error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config_path = source["config_path"];
	        this.repo_root = source["repo_root"];
	        this.remote_url = source["remote_url"];
	        this.vault_file_name = source["vault_file_name"];
	        this.load_error = source["load_error"];
	    }
	}

}

export namespace service {
	
	export class RepoStatus {
	    isGitRepo: boolean;
	    hasRemote: boolean;
	    remoteHasData: boolean;
	    hasLocalVault: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RepoStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isGitRepo = source["isGitRepo"];
	        this.hasRemote = source["hasRemote"];
	        this.remoteHasData = source["remoteHasData"];
	        this.hasLocalVault = source["hasLocalVault"];
	    }
	}

}

export namespace vault {
	
	export class Entry {
	    id: string;
	    name: string;
	    username: string;
	    password: string;
	    note: string;
	    tags: string[];
	    updated_at: number;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.note = source["note"];
	        this.tags = source["tags"];
	        this.updated_at = source["updated_at"];
	    }
	}

}

