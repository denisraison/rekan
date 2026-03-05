import { expect, test } from '@playwright/test';
import path from 'path';
import fs from 'fs';

test.use({ ignoreHTTPSErrors: true, baseURL: 'https://localhost:5173' });

// Create a tiny valid PNG for testing (1x1 red pixel)
const TEST_IMAGE_PATH = '/tmp/test-attach.png';
const PNG_1x1 = Buffer.from(
	'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==',
	'base64',
);
fs.writeFileSync(TEST_IMAGE_PATH, PNG_1x1);

async function loginAsOperador(page: any) {
	await page.goto('/entrar');
	await page.getByLabel('Email').fill('operador@rekan.local');
	await page.getByLabel('Senha').fill('senha1234567');
	await page.getByRole('button', { name: 'Entrar' }).click();
	await page.waitForURL('**/operador**');
	await page.waitForTimeout(2000);
}

async function selectFirstClient(page: any) {
	const clientButton = page.locator('button.text-left.border-b').first();
	await clientButton.click();
	await page.waitForTimeout(1000);
}

/** Attaches a test image via the Galeria option, intercepting the file chooser. */
async function attachImage(page: any) {
	await page.getByRole('button', { name: 'Anexar arquivo' }).click();
	const [fileChooser] = await Promise.all([
		page.waitForEvent('filechooser'),
		page.getByRole('button', { name: 'Galeria' }).click(),
	]);
	await fileChooser.setFiles(TEST_IMAGE_PATH);
	await page.waitForTimeout(300);
}

test.describe('Attach button (Wave 5)', () => {
	test.beforeEach(async ({ page }) => {
		await loginAsOperador(page);
		await selectFirstClient(page);
	});

	test('attach button appears next to input in chat mode', async ({ page }) => {
		const attachBtn = page.getByRole('button', { name: 'Anexar arquivo' });
		await expect(attachBtn).toBeVisible();
	});

	test('attach button appears in generate mode too', async ({ page }) => {
		await page.getByRole('button', { name: 'Post', exact: true }).click();
		await page.waitForTimeout(300);
		const attachBtn = page.getByRole('button', { name: 'Anexar arquivo' });
		await expect(attachBtn).toBeVisible();
	});

	test('tapping attach button shows menu with Galeria, Camera, Video options', async ({ page }) => {
		await page.getByRole('button', { name: 'Anexar arquivo' }).click();
		await expect(page.getByRole('button', { name: 'Galeria' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Camera' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Video' })).toBeVisible();
	});

	test('tapping outside closes the attach menu', async ({ page }) => {
		await page.getByRole('button', { name: 'Anexar arquivo' }).click();
		await expect(page.getByRole('button', { name: 'Galeria' })).toBeVisible();
		await page.getByRole('button', { name: 'Fechar menu' }).click();
		await expect(page.getByRole('button', { name: 'Galeria' })).not.toBeVisible();
	});

	test('selecting a photo shows preview with remove button', async ({ page }) => {
		await attachImage(page);
		// Preview image and remove button should be visible
		await expect(page.getByAltText('Anexo')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Remover anexo' })).toBeVisible();
	});

	test('remove button clears the attachment preview', async ({ page }) => {
		await attachImage(page);
		await expect(page.getByAltText('Anexo')).toBeVisible();
		await page.getByRole('button', { name: 'Remover anexo' }).click();
		await expect(page.getByAltText('Anexo')).not.toBeVisible();
	});

	test('attachment preview persists when typing in the input', async ({ page }) => {
		await attachImage(page);
		await expect(page.getByAltText('Anexo')).toBeVisible();
		const input = page.locator('input[placeholder="Mensagem..."]');
		await input.fill('some text');
		await expect(page.getByAltText('Anexo')).toBeVisible();
	});

	test('switching modes clears the attachment', async ({ page }) => {
		await attachImage(page);
		await expect(page.getByAltText('Anexo')).toBeVisible();
		// Switch to generate mode
		await page.getByRole('button', { name: 'Post', exact: true }).click();
		await page.waitForTimeout(300);
		await expect(page.getByAltText('Anexo')).not.toBeVisible();
	});

	test('generate mode: attach enables the Gerar button', async ({ page }) => {
		await page.getByRole('button', { name: 'Post', exact: true }).click();
		await page.waitForTimeout(300);
		// Gerar should be disabled with no input/selection/attachment
		const gerarBtn = page.getByRole('button', { name: 'Gerar' });
		await expect(gerarBtn).toBeDisabled();
		// Attach an image
		await attachImage(page);
		await expect(gerarBtn).toBeEnabled();
	});

	test('chat mode: attach enables the Enviar button', async ({ page }) => {
		// Enviar should be disabled with no text and no attachment
		const enviarBtn = page.getByRole('button', { name: 'Enviar' });
		await expect(enviarBtn).toBeDisabled();
		// Attach an image
		await attachImage(page);
		await expect(enviarBtn).toBeEnabled();
	});
});
