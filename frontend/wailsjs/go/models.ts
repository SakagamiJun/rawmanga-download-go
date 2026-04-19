export namespace contracts {
	
	export class AppSettings {
	    outputRoot: string;
	    maxConcurrentDownloads: number;
	    retryCount: number;
	    requestTimeoutSec: number;
	    localeMode: string;
	    locale: string;
	    themeMode: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outputRoot = source["outputRoot"];
	        this.maxConcurrentDownloads = source["maxConcurrentDownloads"];
	        this.retryCount = source["retryCount"];
	        this.requestTimeoutSec = source["requestTimeoutSec"];
	        this.localeMode = source["localeMode"];
	        this.locale = source["locale"];
	        this.themeMode = source["themeMode"];
	    }
	}
	export class ChapterItem {
	    id: string;
	    number: number;
	    title: string;
	    releaseDate: string;
	    pageCount: number;
	    localStatus: string;
	    localPath: string;
	    selected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ChapterItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.number = source["number"];
	        this.title = source["title"];
	        this.releaseDate = source["releaseDate"];
	        this.pageCount = source["pageCount"];
	        this.localStatus = source["localStatus"];
	        this.localPath = source["localPath"];
	        this.selected = source["selected"];
	    }
	}
	export class DownloadJob {
	    jobID: string;
	    mangaSlug: string;
	    mangaTitle: string;
	    sourceURL: string;
	    status: string;
	    queuedChapters: number;
	    completedChapters: number;
	    failedChapters: number;
	    createdAt: string;
	    updatedAt: string;
	    lastError: string;
	    maxConcurrentPages: number;
	
	    static createFrom(source: any = {}) {
	        return new DownloadJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobID = source["jobID"];
	        this.mangaSlug = source["mangaSlug"];
	        this.mangaTitle = source["mangaTitle"];
	        this.sourceURL = source["sourceURL"];
	        this.status = source["status"];
	        this.queuedChapters = source["queuedChapters"];
	        this.completedChapters = source["completedChapters"];
	        this.failedChapters = source["failedChapters"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.lastError = source["lastError"];
	        this.maxConcurrentPages = source["maxConcurrentPages"];
	    }
	}
	export class LibraryManga {
	    id: string;
	    title: string;
	    relativePath: string;
	    coverImageURL: string;
	    chapterCount: number;
	    pageCount: number;
	    lastUpdated: string;
	
	    static createFrom(source: any = {}) {
	        return new LibraryManga(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.relativePath = source["relativePath"];
	        this.coverImageURL = source["coverImageURL"];
	        this.chapterCount = source["chapterCount"];
	        this.pageCount = source["pageCount"];
	        this.lastUpdated = source["lastUpdated"];
	    }
	}
	export class LocalChapterState {
	    chapterID: string;
	    chapterNumber: number;
	    title: string;
	    status: string;
	    localPath: string;
	    localPageCount: number;
	    expectedPageCount: number;
	
	    static createFrom(source: any = {}) {
	        return new LocalChapterState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapterID = source["chapterID"];
	        this.chapterNumber = source["chapterNumber"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.localPath = source["localPath"];
	        this.localPageCount = source["localPageCount"];
	        this.expectedPageCount = source["expectedPageCount"];
	    }
	}
	export class LocalSummary {
	    notDownloaded: number;
	    partial: number;
	    complete: number;
	    missing: number;
	
	    static createFrom(source: any = {}) {
	        return new LocalSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.notDownloaded = source["notDownloaded"];
	        this.partial = source["partial"];
	        this.complete = source["complete"];
	        this.missing = source["missing"];
	    }
	}
	export class ParsedMangaResult {
	    sourceURL: string;
	    slug: string;
	    title: string;
	    coverURL: string;
	    chapters: ChapterItem[];
	    localSummary: LocalSummary;
	    profileCacheHit: boolean;
	    algorithmProfile: string;
	
	    static createFrom(source: any = {}) {
	        return new ParsedMangaResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceURL = source["sourceURL"];
	        this.slug = source["slug"];
	        this.title = source["title"];
	        this.coverURL = source["coverURL"];
	        this.chapters = this.convertValues(source["chapters"], ChapterItem);
	        this.localSummary = this.convertValues(source["localSummary"], LocalSummary);
	        this.profileCacheHit = source["profileCacheHit"];
	        this.algorithmProfile = source["algorithmProfile"];
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
	export class QueueDownloadRequest {
	    sourceURL: string;
	    mangaSlug: string;
	    title: string;
	    chapterIDs: string[];
	    outputRoot: string;
	
	    static createFrom(source: any = {}) {
	        return new QueueDownloadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceURL = source["sourceURL"];
	        this.mangaSlug = source["mangaSlug"];
	        this.title = source["title"];
	        this.chapterIDs = source["chapterIDs"];
	        this.outputRoot = source["outputRoot"];
	    }
	}
	export class ReaderPage {
	    id: string;
	    chapterID: string;
	    chapterTitle: string;
	    pageIndex: number;
	    fileName: string;
	    sourceURL: string;
	
	    static createFrom(source: any = {}) {
	        return new ReaderPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.chapterID = source["chapterID"];
	        this.chapterTitle = source["chapterTitle"];
	        this.pageIndex = source["pageIndex"];
	        this.fileName = source["fileName"];
	        this.sourceURL = source["sourceURL"];
	    }
	}
	export class ReaderChapter {
	    id: string;
	    title: string;
	    number: number;
	    startPage: number;
	    pageCount: number;
	    pages: ReaderPage[];
	    localPath: string;
	    completedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ReaderChapter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.number = source["number"];
	        this.startPage = source["startPage"];
	        this.pageCount = source["pageCount"];
	        this.pages = this.convertValues(source["pages"], ReaderPage);
	        this.localPath = source["localPath"];
	        this.completedAt = source["completedAt"];
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
	export class ReaderManifest {
	    mangaID: string;
	    title: string;
	    coverImageURL: string;
	    totalPages: number;
	    chapters: ReaderChapter[];
	
	    static createFrom(source: any = {}) {
	        return new ReaderManifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mangaID = source["mangaID"];
	        this.title = source["title"];
	        this.coverImageURL = source["coverImageURL"];
	        this.totalPages = source["totalPages"];
	        this.chapters = this.convertValues(source["chapters"], ReaderChapter);
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

