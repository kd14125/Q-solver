export namespace config {
	
	export class Config {
	    apiKey?: string;
	    provider?: string;
	    model?: string;
	    baseURL?: string;
	    prompt?: string;
	    theme?: string;
	    opacity: number;
	    noCompression?: boolean;
	    compressionQuality?: number;
	    sharpening?: number;
	    grayscale?: boolean;
	    keepContext?: boolean;
	    interruptThinking?: boolean;
	    screenshotMode?: string;
	    resumePath?: string;
	    resumeContent?: string;
	    useMarkdownResume?: boolean;
	    shortcuts?: Record<string, shortcut.KeyBinding>;
	    temperature?: number;
	    topP?: number;
	    topK?: number;
	    maxTokens?: number;
	    thinkingBudget?: number;
	    assistantModel?: string;
	    useLiveApi?: boolean;
	    sttEnabled?: boolean;
	    sttAPIKey?: string;
	    sttBaseURL?: string;
	    sttModel?: string;
	    sttLanguage?: string;
	    voiceReply?: boolean;
	    realtimeEnabled: boolean;
	    realtimeAPIKey?: string;
	    realtimeWorkspaceID?: string;
	    realtimeRegion?: string;
	    realtimeBaseURL?: string;
	    realtimeModel?: string;
	    realtimePrompt?: string;
	    realtimeTemperature: number;
	    realtimeTopP: number;
	    realtimeTopK: number;
	    realtimeMaxTokens: number;
	    realtimeVADType?: string;
	    realtimeVADThreshold: number;
	    realtimeSilenceDurationMs: number;
	    ragEnabled: boolean;
	    ragRetrievalMode?: string;
	    ragEmbeddingModel?: string;
	    ragEmbeddingDimensions: number;
	    ragAPIKey?: string;
	    ragWorkspaceID?: string;
	    ragRegion?: string;
	    ragBaseURL?: string;
	    ragTopK: number;
	    ragMaxContextChars: number;
	    windowWidth?: number;
	    windowHeight?: number;
	    aiFontSize?: number;
	    codeWrap?: boolean;
	    aiTextTransparency?: number;
	    aiTextColor?: string;
	    hideTopBar?: boolean;
	    hideHistoryPanel?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKey = source["apiKey"];
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.baseURL = source["baseURL"];
	        this.prompt = source["prompt"];
	        this.theme = source["theme"];
	        this.opacity = source["opacity"];
	        this.noCompression = source["noCompression"];
	        this.compressionQuality = source["compressionQuality"];
	        this.sharpening = source["sharpening"];
	        this.grayscale = source["grayscale"];
	        this.keepContext = source["keepContext"];
	        this.interruptThinking = source["interruptThinking"];
	        this.screenshotMode = source["screenshotMode"];
	        this.resumePath = source["resumePath"];
	        this.resumeContent = source["resumeContent"];
	        this.useMarkdownResume = source["useMarkdownResume"];
	        this.shortcuts = this.convertValues(source["shortcuts"], shortcut.KeyBinding, true);
	        this.temperature = source["temperature"];
	        this.topP = source["topP"];
	        this.topK = source["topK"];
	        this.maxTokens = source["maxTokens"];
	        this.thinkingBudget = source["thinkingBudget"];
	        this.assistantModel = source["assistantModel"];
	        this.useLiveApi = source["useLiveApi"];
	        this.sttEnabled = source["sttEnabled"];
	        this.sttAPIKey = source["sttAPIKey"];
	        this.sttBaseURL = source["sttBaseURL"];
	        this.sttModel = source["sttModel"];
	        this.sttLanguage = source["sttLanguage"];
	        this.voiceReply = source["voiceReply"];
	        this.realtimeEnabled = source["realtimeEnabled"];
	        this.realtimeAPIKey = source["realtimeAPIKey"];
	        this.realtimeWorkspaceID = source["realtimeWorkspaceID"];
	        this.realtimeRegion = source["realtimeRegion"];
	        this.realtimeBaseURL = source["realtimeBaseURL"];
	        this.realtimeModel = source["realtimeModel"];
	        this.realtimePrompt = source["realtimePrompt"];
	        this.realtimeTemperature = source["realtimeTemperature"];
	        this.realtimeTopP = source["realtimeTopP"];
	        this.realtimeTopK = source["realtimeTopK"];
	        this.realtimeMaxTokens = source["realtimeMaxTokens"];
	        this.realtimeVADType = source["realtimeVADType"];
	        this.realtimeVADThreshold = source["realtimeVADThreshold"];
	        this.realtimeSilenceDurationMs = source["realtimeSilenceDurationMs"];
	        this.ragEnabled = source["ragEnabled"];
	        this.ragRetrievalMode = source["ragRetrievalMode"];
	        this.ragEmbeddingModel = source["ragEmbeddingModel"];
	        this.ragEmbeddingDimensions = source["ragEmbeddingDimensions"];
	        this.ragAPIKey = source["ragAPIKey"];
	        this.ragWorkspaceID = source["ragWorkspaceID"];
	        this.ragRegion = source["ragRegion"];
	        this.ragBaseURL = source["ragBaseURL"];
	        this.ragTopK = source["ragTopK"];
	        this.ragMaxContextChars = source["ragMaxContextChars"];
	        this.windowWidth = source["windowWidth"];
	        this.windowHeight = source["windowHeight"];
	        this.aiFontSize = source["aiFontSize"];
	        this.codeWrap = source["codeWrap"];
	        this.aiTextTransparency = source["aiTextTransparency"];
	        this.aiTextColor = source["aiTextColor"];
	        this.hideTopBar = source["hideTopBar"];
	        this.hideHistoryPanel = source["hideHistoryPanel"];
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

export namespace rag {
	
	export class Document {
	    id: number;
	    name: string;
	    path?: string;
	    kind: string;
	    status: string;
	    chunkCount: number;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Document(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.status = source["status"];
	        this.chunkCount = source["chunkCount"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ImportResult {
	    path: string;
	    document: Document;
	    warning?: string;
	    duplicated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.document = this.convertValues(source["document"], Document);
	        this.warning = source["warning"];
	        this.duplicated = source["duplicated"];
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
	export class IndexResult {
	    total: number;
	    indexed: number;
	    warning?: string;
	
	    static createFrom(source: any = {}) {
	        return new IndexResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.indexed = source["indexed"];
	        this.warning = source["warning"];
	    }
	}
	export class QAEntry {
	    id: number;
	    question: string;
	    answer: string;
	    status: string;
	    warning?: string;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new QAEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.question = source["question"];
	        this.answer = source["answer"];
	        this.status = source["status"];
	        this.warning = source["warning"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class SearchHit {
	    id: number;
	    kind: string;
	    title: string;
	    content: string;
	    source: string;
	    score: number;
	    localRank?: number;
	    vectorRank?: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.source = source["source"];
	        this.score = source["score"];
	        this.localRank = source["localRank"];
	        this.vectorRank = source["vectorRank"];
	    }
	}
	export class SearchResult {
	    mode: string;
	    hits: SearchHit[];
	    warning?: string;
	    durationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.hits = this.convertValues(source["hits"], SearchHit);
	        this.warning = source["warning"];
	        this.durationMs = source["durationMs"];
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
	export class SearchTestResult {
	    local: SearchResult;
	    api: SearchResult;
	    hybrid: SearchResult;
	
	    static createFrom(source: any = {}) {
	        return new SearchTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.local = this.convertValues(source["local"], SearchResult);
	        this.api = this.convertValues(source["api"], SearchResult);
	        this.hybrid = this.convertValues(source["hybrid"], SearchResult);
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
	export class Snapshot {
	    documents: Document[];
	    qaEntries: QAEntry[];
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.documents = this.convertValues(source["documents"], Document);
	        this.qaEntries = this.convertValues(source["qaEntries"], QAEntry);
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

export namespace screen {
	
	export class PreviewResult {
	    imgBytes: number[];
	    base64: string;
	    size: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imgBytes = source["imgBytes"];
	        this.base64 = source["base64"];
	        this.size = source["size"];
	    }
	}

}

export namespace shortcut {
	
	export class KeyBinding {
	    vkCode: string;
	    keyName: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vkCode = source["vkCode"];
	        this.keyName = source["keyName"];
	    }
	}

}

