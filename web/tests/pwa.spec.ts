import { expect, test } from '@playwright/test';

test('service worker registers and manifest is linked', async ({ page }) => {
	await page.goto('/operador');

	await expect(async () => {
		const count = await page.evaluate(async () => {
			const regs = await navigator.serviceWorker.getRegistrations();
			return regs.length;
		});
		expect(count).toBeGreaterThan(0);
	}).toPass({ timeout: 10_000 });

	const manifest = page.locator('link[rel="manifest"]');
	await expect(manifest).toHaveAttribute('href', '/manifest.webmanifest');
});
