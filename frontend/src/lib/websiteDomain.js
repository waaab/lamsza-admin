const tagRE = /<[^>]*>/g;

const multiSuffix = new Set(["co.uk", "org.uk", "com.au"]);

export function canonicalDomain(raw) {
	const s = raw.trim();
	if (s === "") {
		return "";
	}
	let urlStr = s;
	if (!s.includes("://")) {
		urlStr = "https://" + s;
	}
	let hostname;
	try {
		hostname = new URL(urlStr).hostname.toLowerCase();
	} catch {
		return "";
	}
	if (hostname === "") {
		return "";
	}
	let host = hostname.startsWith("www.") ? hostname.slice(4) : hostname;
	if (host.includes(" ")) {
		return "";
	}
	const labels = host.split(".");
	if (labels.length < 2 || labels[0] === "") {
		return "";
	}
	const suffix = labels[labels.length - 2] + "." + labels[labels.length - 1];
	if (multiSuffix.has(suffix)) {
		if (labels.length < 3) {
			return "";
		}
		return labels.slice(-3).join(".");
	}
	return labels.slice(-2).join(".");
}

export function plainText(raw, max) {
	const s = raw.replace(tagRE, "").trim();
	if (s === "") {
		return "";
	}
	if ([...s].length > max) {
		return "";
	}
	return s;
}
