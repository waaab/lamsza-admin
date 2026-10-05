import { hoursConfigured } from "./entryHours.js";

const DELIVERY_CATEGORIES = new Set(["Étterem", "Kávézó", "Cukrászda", "Pékség", "Söröző"]);

export function offersDelivery(entry) {
	if (String(entry?.name ?? "").trim() === "Lámsza.com") return false;
	return DELIVERY_CATEGORIES.has(String(entry?.category ?? "").trim());
}

export function showListingPhotos(entry) {
	return Boolean(entry?.claimed) && Boolean(entry?.verified) && Array.isArray(entry?.photos) && entry.photos.length > 0;
}

export function showListingHours(entry) {
	return Boolean(entry?.hours_enabled);
}

export function showListingDeliveryHours(entry) {
	return Boolean(entry?.delivery_enabled) && offersDelivery(entry);
}

export function showListingRatings(entry) {
	return Boolean(entry?.claimed) && Boolean(entry?.ratings_enabled);
}

export function showListingTodayHours(entry) {
	return showListingHours(entry);
}

export function showListingPhone(entry) {
	return Boolean(entry?.claimed) && String(entry?.phone ?? "").trim() !== "";
}

export function showListingSocial(entry) {
	return Boolean(entry?.claimed) && Array.isArray(entry?.social_links) && entry.social_links.some((row) => String(row?.url ?? "").trim() !== "");
}

export function showListingWebsite(entry) {
	return String(entry?.url ?? "").trim() !== "";
}

export function showListingLanguages(entry) {
	return Array.isArray(entry?.languages) && entry.languages.some((row) => String(row ?? "").trim() !== "");
}

export function listingTextExpanded(entry) {
	return Boolean(entry?.claimed) && Boolean(entry?.verified);
}

export { hoursConfigured };
