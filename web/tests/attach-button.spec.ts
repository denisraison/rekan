import { expect, test } from '@playwright/test';
import fs from 'fs';
import { loginAsOperador, selectFirstClient, switchToGenerateMode } from './helpers';

const TEST_IMAGE_PATH = '/tmp/test-attach.png';
fs.writeFileSync(
  TEST_IMAGE_PATH,
  Buffer.from(
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==',
    'base64',
  ),
);

async function attachImage(page: any) {
  await page.getByRole('button', { name: 'Anexar arquivo' }).click();
  const [fileChooser] = await Promise.all([
    page.waitForEvent('filechooser'),
    page.getByRole('button', { name: 'Galeria' }).click(),
  ]);
  await fileChooser.setFiles(TEST_IMAGE_PATH);
  await page.getByAltText('Anexo').waitFor();
}

test.describe('Attach button (Wave 5)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsOperador(page);
    await selectFirstClient(page);
  });

  test('attach button appears in both modes', async ({ page }) => {
    await expect(page.getByRole('button', { name: 'Anexar arquivo' })).toBeVisible();
    await switchToGenerateMode(page);
    await expect(page.getByRole('button', { name: 'Anexar arquivo' })).toBeVisible();
  });

  test('menu shows Galeria and Camera, closes on backdrop', async ({ page }) => {
    await page.getByRole('button', { name: 'Anexar arquivo' }).click();
    await expect(page.getByRole('button', { name: 'Galeria' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Camera' })).toBeVisible();
    await page.getByRole('button', { name: 'Fechar menu' }).click();
    await expect(page.getByRole('button', { name: 'Galeria' })).not.toBeVisible();
  });

  test('selecting a photo shows preview, remove clears it', async ({ page }) => {
    await attachImage(page);
    await expect(page.getByRole('button', { name: 'Remover anexo' })).toBeVisible();
    await page.getByRole('button', { name: 'Remover anexo' }).click();
    await expect(page.getByAltText('Anexo')).not.toBeVisible();
  });

  test('preview persists when typing', async ({ page }) => {
    await attachImage(page);
    await page.locator('input[placeholder="Mensagem..."]').fill('some text');
    await expect(page.getByAltText('Anexo')).toBeVisible();
  });

  test('switching modes clears attachment', async ({ page }) => {
    await attachImage(page);
    await switchToGenerateMode(page);
    await expect(page.getByAltText('Anexo')).not.toBeVisible();
  });

  test('attach enables Gerar in generate mode', async ({ page }) => {
    await switchToGenerateMode(page);
    const gerarBtn = page.getByRole('button', { name: 'Gerar' });
    await expect(gerarBtn).toBeDisabled();
    await attachImage(page);
    await expect(gerarBtn).toBeEnabled();
  });

  test('attach enables Enviar in chat mode', async ({ page }) => {
    await expect(page.getByRole('button', { name: 'Enviar' })).toBeDisabled();
    await attachImage(page);
    await expect(page.getByRole('button', { name: 'Enviar' })).toBeEnabled();
  });
});
