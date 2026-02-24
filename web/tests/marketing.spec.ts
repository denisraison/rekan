import { expect, test } from '@playwright/test';

test('page loads with title', async ({ page }) => {
	await page.goto('/');
	await expect(page).toHaveTitle(/Rekan/);
});

test('nav links are visible', async ({ page }) => {
	await page.goto('/');
	const nav = page.locator('nav');
	await expect(nav).toBeVisible();
	await expect(nav.getByRole('link', { name: 'Como funciona' })).toBeVisible();
	await expect(nav.getByRole('link', { name: 'Exemplos' })).toBeVisible();
});

test('hero section renders', async ({ page }) => {
	await page.goto('/');
	await expect(page.locator('h1')).toContainText('Post pronto');
});

test('examples section has 3 phone frames', async ({ page }) => {
	await page.goto('/');
	const phones = page.locator('#exemplos .phone-slot');
	await expect(phones).toHaveCount(3);
});

test('pricing shows all three tiers', async ({ page }) => {
	await page.goto('/');
	const pricing = page.locator('#preco');
	await expect(pricing).toContainText('69,90');
	await expect(pricing).toContainText('108,90');
	await expect(pricing).toContainText('249,90');
});

test('pricing shows social media manager anchor', async ({ page }) => {
	await page.goto('/');
	await expect(page.locator('#preco')).toContainText('R$590');
});

test('CTA links to WhatsApp', async ({ page }) => {
	await page.goto('/');
	const cta = page.locator('.hero-cta a.btn-primary');
	await expect(cta).toHaveAttribute('href', /wa\.me/);
});
