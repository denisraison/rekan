import { test as setup } from '@playwright/test';

const STORAGE_STATE = 'tests/.auth/operador.json';

setup('authenticate as operador', async ({ page }) => {
  await page.goto('/entrar');
  await page.getByLabel('Email').fill('operador@rekan.local');
  await page.getByLabel('Senha').fill('senha1234567');
  await page.getByRole('button', { name: 'Entrar' }).click();
  await page.locator('button.text-left.border-b').first().waitFor();
  await page.context().storageState({ path: STORAGE_STATE });
});
