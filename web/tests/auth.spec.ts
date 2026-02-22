import { expect, test } from '@playwright/test';

test('login page loads with Google button', async ({ page }) => {
	await page.goto('/login');
	await expect(page.getByRole('button', { name: /Google/ })).toBeVisible();
});

test('unauthenticated /dashboard redirects to /login', async ({ page }) => {
	await page.goto('/dashboard');
	await page.waitForURL('/login');
	await expect(page).toHaveURL('/login');
});
