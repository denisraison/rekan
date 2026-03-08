import { expect, test } from '@playwright/test';

test.describe('unauthenticated', () => {
	test.use({ storageState: { cookies: [], origins: [] } });

	test('login page loads with email and password fields', async ({ page }) => {
		await page.goto('/entrar');
		await expect(page.getByLabel('Email')).toBeVisible();
		await expect(page.getByLabel('Senha')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Entrar' })).toBeVisible();
	});

	test('unauthenticated /operador redirects to /entrar', async ({ page }) => {
		await page.goto('/operador');
		await page.waitForURL('/entrar');
		await expect(page).toHaveURL('/entrar');
	});
});

test.describe('authenticated', () => {
	// Uses default storageState (authenticated operador)

	test('authenticated user visiting /entrar is redirected to /operador', async ({ page }) => {
		// Mock WhatsApp stream to avoid QR code overlay
		await page.route('**/api/whatsapp/stream', (route) => {
			route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				headers: { 'Cache-Control': 'no-cache', 'Connection': 'keep-alive' },
				body: 'data: {"connected":true,"qr":""}\n\n',
			});
		});
		await page.goto('/entrar');
		await page.waitForURL('/operador');
		await expect(page).toHaveURL('/operador');
	});
});
