import { test } from 'node:test';
import assert from 'node:assert/strict';
import {
	showListingPhotos,
	showListingHours,
	showListingDeliveryHours,
	showListingRatings,
	showListingPhone,
	showListingSocial,
	listingTextExpanded
} from '../src/lib/entryPublicExtras.js';

test('showListingPhotos is true only when claimed, verified, and a photo is stored', () => {
	assert.equal(showListingPhotos({ verified: true, claimed: false, photos: [{ id: 1 }] }), false);
	assert.equal(showListingPhotos({ verified: false, claimed: true, photos: [{ id: 1 }] }), false);
	assert.equal(showListingPhotos({ verified: true, claimed: true, photos: [{ id: 1 }] }), true);
	assert.equal(showListingPhotos({ verified: true, claimed: true, photos: [] }), false);
});

test('showListingHours follows hours_enabled even when gazdátlan or the week is empty', () => {
	assert.equal(showListingHours({ claimed: false, hours_enabled: true, hours: {} }), true);
	assert.equal(showListingHours({ claimed: true, hours_enabled: false, hours: { mon: { open: '09:00', close: '17:00' } } }), false);
});

test('showListingDeliveryHours requires the switch and a delivery category', () => {
	assert.equal(showListingDeliveryHours({
		delivery_enabled: true,
		category: 'Étterem',
		name: 'Példa'
	}), true);
	assert.equal(showListingDeliveryHours({
		delivery_enabled: true,
		category: 'Étterem',
		name: 'Lámsza.com'
	}), false);
	assert.equal(showListingDeliveryHours({
		delivery_enabled: true,
		category: 'Kávézó',
		name: 'Példa'
	}), true);
	assert.equal(showListingDeliveryHours({
		delivery_enabled: true,
		category: 'Bolt',
		name: 'Példa'
	}), false);
});

test('phone and social show only when claimed and stored', () => {
	assert.equal(showListingPhone({ claimed: false, phone: '0700000000' }), false);
	assert.equal(showListingPhone({ claimed: true, phone: '0700000000' }), true);
	assert.equal(showListingPhone({ claimed: true, phone: '  ' }), false);
	assert.equal(showListingSocial({ claimed: true, social_links: [{ label: 'Facebook', url: 'https://facebook.com/a' }] }), true);
	assert.equal(showListingSocial({ claimed: false, social_links: [{ label: 'Facebook', url: 'https://facebook.com/a' }] }), false);
});

test('listingTextExpanded is true only when claimed and verified', () => {
	assert.equal(listingTextExpanded({ verified: true, claimed: false }), false);
	assert.equal(listingTextExpanded({ verified: true, claimed: true }), true);
	assert.equal(listingTextExpanded({ verified: false, claimed: true }), false);
});

test('showListingRatings returns false for unclaimed entry', () => {
	const entry = { claimed: false, ratings_enabled: true };
	assert.equal(showListingRatings(entry), false);
});

test('showListingRatings returns false for claimed entry without ratings_enabled', () => {
	const entry = { claimed: true, ratings_enabled: false };
	assert.equal(showListingRatings(entry), false);
});

test('showListingRatings returns true for claimed entry with ratings_enabled', () => {
	const entry = { claimed: true, ratings_enabled: true };
	assert.equal(showListingRatings(entry), true);
});
