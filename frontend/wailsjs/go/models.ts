export namespace domain {

	export class ActiveLiveNowSession {
	    destinationID: string;
	    destinationName: string;
	    platform: string;
	    accountIdentifier: string;
	    credentialKey: string;
	    streamURL: string;
	    streamTitle: string;
	    // Go type: time
	    startedAt: any;

	    static createFrom(source: any = {}) {
	        return new ActiveLiveNowSession(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.destinationID = source["destinationID"];
	        this.destinationName = source["destinationName"];
	        this.platform = source["platform"];
	        this.accountIdentifier = source["accountIdentifier"];
	        this.credentialKey = source["credentialKey"];
	        this.streamURL = source["streamURL"];
	        this.streamTitle = source["streamTitle"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
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
	export class Announcement {
	    streamTitle: string;
	    streamURL: string;
	    category: string;
	    message: string;
	    hashtags: string;
	    announcementImageURL: string;
	    announcementImageDataURL: string;
	    blueskyCardThumbnailURL: string;
	    blueskyCardThumbnailDataURL: string;
	    destinationGroupID: string;
	    destinationIDs: string[];
	    // Go type: time
	    createdAt: any;

	    static createFrom(source: any = {}) {
	        return new Announcement(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.streamTitle = source["streamTitle"];
	        this.streamURL = source["streamURL"];
	        this.category = source["category"];
	        this.message = source["message"];
	        this.hashtags = source["hashtags"];
	        this.announcementImageURL = source["announcementImageURL"];
	        this.announcementImageDataURL = source["announcementImageDataURL"];
	        this.blueskyCardThumbnailURL = source["blueskyCardThumbnailURL"];
	        this.blueskyCardThumbnailDataURL = source["blueskyCardThumbnailDataURL"];
	        this.destinationGroupID = source["destinationGroupID"];
	        this.destinationIDs = source["destinationIDs"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	export class AnnouncementProfile {
	    id: string;
	    name: string;
	    description: string;
	    defaultStreamTitle: string;
	    defaultStreamURL: string;
	    defaultCategory: string;
	    defaultMessage: string;
	    defaultHashtags: string;
	    defaultAnnouncementImageURL: string;
	    defaultAnnouncementImageDataURL: string;
	    defaultBlueskyCardThumbnailURL: string;
	    defaultBlueskyCardThumbnailDataURL: string;
	    defaultGroupID: string;
	    // Go type: time
	    lastUsedAt?: any;
	    useCount: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new AnnouncementProfile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.defaultStreamTitle = source["defaultStreamTitle"];
	        this.defaultStreamURL = source["defaultStreamURL"];
	        this.defaultCategory = source["defaultCategory"];
	        this.defaultMessage = source["defaultMessage"];
	        this.defaultHashtags = source["defaultHashtags"];
	        this.defaultAnnouncementImageURL = source["defaultAnnouncementImageURL"];
	        this.defaultAnnouncementImageDataURL = source["defaultAnnouncementImageDataURL"];
	        this.defaultBlueskyCardThumbnailURL = source["defaultBlueskyCardThumbnailURL"];
	        this.defaultBlueskyCardThumbnailDataURL = source["defaultBlueskyCardThumbnailDataURL"];
	        this.defaultGroupID = source["defaultGroupID"];
	        this.lastUsedAt = this.convertValues(source["lastUsedAt"], null);
	        this.useCount = source["useCount"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class AppSettings {
	    defaultStreamURL: string;
	    defaultHashtags: string;
	    duplicateProtectionEnabled: boolean;
	    duplicateWindowMinutes: number;
	    selectedDestinationGroupID: string;

	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultStreamURL = source["defaultStreamURL"];
	        this.defaultHashtags = source["defaultHashtags"];
	        this.duplicateProtectionEnabled = source["duplicateProtectionEnabled"];
	        this.duplicateWindowMinutes = source["duplicateWindowMinutes"];
	        this.selectedDestinationGroupID = source["selectedDestinationGroupID"];
	    }
	}
	export class CredentialCheckResult {
	    platform: string;
	    state: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new CredentialCheckResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.state = source["state"];
	        this.message = source["message"];
	    }
	}
	export class Destination {
	    id: string;
	    platform: string;
	    name: string;
	    enabled: boolean;
	    template: string;
	    configJSON: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new Destination(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.template = source["template"];
	        this.configJSON = source["configJSON"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class DestinationGroup {
	    id: string;
	    name: string;
	    description: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new DestinationGroup(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class ExecutionResult {
	    destinationID: string;
	    destinationName: string;
	    platform: string;
	    state: string;
	    message: string;
	    content: string;
	    receipt?: PublishReceipt;

	    static createFrom(source: any = {}) {
	        return new ExecutionResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.destinationID = source["destinationID"];
	        this.destinationName = source["destinationName"];
	        this.platform = source["platform"];
	        this.state = source["state"];
	        this.message = source["message"];
	        this.content = source["content"];
	        this.receipt = source["receipt"] ? new PublishReceipt(source["receipt"]) : undefined;
	    }
	}
	export class PublishReceipt {
	    provider: string;
	    id?: string;
	    uri?: string;
	    cid?: string;

	    static createFrom(source: any = {}) {
	        return new PublishReceipt(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.id = source["id"];
	        this.uri = source["uri"];
	        this.cid = source["cid"];
	    }
	}
	export class ExecutionSummary {
	    mode: string;
	    status: string;
	    testModeActive: boolean;
	    results: ExecutionResult[];
	    totalCount: number;
	    successCount: number;
	    failedCount: number;
	    skippedCount: number;
	    validationErrorCount: number;
	    requiresDuplicateConfirmation: boolean;
	    duplicateWarningMessage: string;

	    static createFrom(source: any = {}) {
	        return new ExecutionSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.testModeActive = source["testModeActive"];
	        this.results = this.convertValues(source["results"], ExecutionResult);
	        this.totalCount = source["totalCount"];
	        this.successCount = source["successCount"];
	        this.failedCount = source["failedCount"];
	        this.skippedCount = source["skippedCount"];
	        this.validationErrorCount = source["validationErrorCount"];
	        this.requiresDuplicateConfirmation = source["requiresDuplicateConfirmation"];
	        this.duplicateWarningMessage = source["duplicateWarningMessage"];
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
	export class LogEntry {
	    // Go type: time
	    timestamp: any;
	    destination: string;
	    action: string;
	    status: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.destination = source["destination"];
	        this.action = source["action"];
	        this.status = source["status"];
	        this.message = source["message"];
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
	export class PreviewItem {
	    destinationID: string;
	    destinationName: string;
	    platform: string;
	    previewLabel: string;
	    content: string;
	    characterCount: number;
	    validationState: string;
	    validationNotes: string[];
	    linkPreview?: LinkPreview;

	    static createFrom(source: any = {}) {
	        return new PreviewItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.destinationID = source["destinationID"];
	        this.destinationName = source["destinationName"];
	        this.platform = source["platform"];
	        this.previewLabel = source["previewLabel"];
	        this.content = source["content"];
	        this.characterCount = source["characterCount"];
	        this.validationState = source["validationState"];
	        this.validationNotes = source["validationNotes"];
	        this.linkPreview = this.convertValues(source["linkPreview"], LinkPreview);
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
	export class LinkPreview {
	    url: string;
	    title: string;
	    description: string;
	    imageURL: string;
	    siteName: string;

	    static createFrom(source: any = {}) {
	        return new LinkPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.imageURL = source["imageURL"];
	        this.siteName = source["siteName"];
	    }
	}

}
