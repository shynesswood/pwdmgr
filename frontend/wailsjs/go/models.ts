export namespace config {
	
	export class Snapshot {
	    config_path: string;
	    repo_root: string;
	    remote_url: string;
	    git_client: string;
	    vault_file_name: string;
	    load_error?: string;
	    search_paths?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config_path = source["config_path"];
	        this.repo_root = source["repo_root"];
	        this.remote_url = source["remote_url"];
	        this.git_client = source["git_client"];
	        this.vault_file_name = source["vault_file_name"];
	        this.load_error = source["load_error"];
	        this.search_paths = source["search_paths"];
	    }
	}

}

export namespace service {
	
	export class RepoStatus {
	    isGitRepo: boolean;
	    hasRemote: boolean;
	    remoteHasData: boolean;
	    hasLocalVault: boolean;
	    hasUncommitted: boolean;
	    currentBranch: string;
	    remoteURL: string;
	
	    static createFrom(source: any = {}) {
	        return new RepoStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isGitRepo = source["isGitRepo"];
	        this.hasRemote = source["hasRemote"];
	        this.remoteHasData = source["remoteHasData"];
	        this.hasLocalVault = source["hasLocalVault"];
	        this.hasUncommitted = source["hasUncommitted"];
	        this.currentBranch = source["currentBranch"];
	        this.remoteURL = source["remoteURL"];
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
	    deleted_at?: number;
	    space_id?: string;
	
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
	        this.deleted_at = source["deleted_at"];
	        this.space_id = source["space_id"];
	    }
	}
	export class Space {
	    id: string;
	    name: string;
	    created_at: number;
	    updated_at: number;
	    deleted_at?: number;
	
	    static createFrom(source: any = {}) {
	        return new Space(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	        this.deleted_at = source["deleted_at"];
	    }
	}

}

