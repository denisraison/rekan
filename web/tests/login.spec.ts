import { expect, test } from '@playwright/test';

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
