import { test, expect, type Page } from '@playwright/test';
import { createServer } from 'node:http';
import { randomBytes } from 'node:crypto';

const email = process.env.NEXTROLE_TEST_ADMIN || 'admin@nextrole.local';
const password = process.env.NEXTROLE_TEST_PASSWORD;
test.use({ actionTimeout: 15000 });
test.skip(!password, 'NEXTROLE_TEST_PASSWORD is required for browser integration checks');

async function administrator(page: Page) {
  const response = await page.request.post('/api/v1/auth/login', { data: { email, password } });
  expect(response.ok()).toBeTruthy();
}
async function noHorizontalOverflow(page: Page, label: string) {
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 2);
  expect(overflow, label).toBe(false);
}

// This is a real HTTP upstream fixture reached by the Go connector. Application
// routes are not intercepted, and no successful request to Work24 is claimed.
const officialFixtures: Record<string, string> = {
  'work24-occupation-summary': '<jobSum><jobCd>133301</jobCd><jobSmclNm>합성 직업 예시</jobSmclNm><jobSum>선택하지 않은 설명</jobSum><sal>선택하지 않은 급여</sal><unknown>보관하지 않는 원문</unknown></jobSum>',
  'work24-ncs': JSON.stringify({ result: { '합성 능력단위': { ablt_unit: '2001020201_23v1', job_sdvn: '합성 능력단위', ablt_def: '선택하지 않은 정의', knwg_tchn_attd: '선택하지 않은 지식', unknown: '보관하지 않는 원문' } } }),
  'work24-jobs': '<wantedRoot><total>1</total><wanted><wantedAuthNo>E2E-J1</wantedAuthNo><title>합성 Python Kubernetes 공고</title><company>합성 테스트 기관</company><wantedInfoUrl>https://example.test/jobs/E2E-J1</wantedInfoUrl><sal>선택하지 않은 급여</sal><skills>Python</skills><unknown>보관하지 않는 원문</unknown></wanted></wantedRoot>',
  'work24-training': '<HRDNet><scn_cnt>1</scn_cnt><srchList><scn_list><trprId>E2E-C1</trprId><trprDegr>3</trprDegr><title>합성 훈련 과정</title><subTitle>합성 훈련기관</subTitle><titleLink>https://example.test/course/3</titleLink><courseMan>선택하지 않은 수강비</courseMan><eiEmplCnt3Gt10>Null</eiEmplCnt3Gt10><unknown>보관하지 않는 원문</unknown></scn_list></srchList></HRDNet>',
};

test('고용24 4개 공식 프리셋: 필수 필드·조회 조건·선별 정규화 미리보기', async ({ page }) => {
  await administrator(page);
  const errors: string[] = [];
  page.on('pageerror', e => errors.push(e.message));
  const token = randomBytes(18).toString('hex');
  const called: string[] = [];
  const upstream = createServer((request, response) => {
    const url = new URL(request.url || '/', 'http://localhost');
    const preset = url.pathname.slice(1);
    if (url.searchParams.get('authKey') !== token || !officialFixtures[preset]) {
      response.writeHead(400); response.end('fixture request rejected'); return;
    }
    called.push(preset);
    response.setHeader('Content-Type', preset === 'work24-ncs' ? 'application/json' : 'application/xml');
    response.end(officialFixtures[preset]);
  });
  await new Promise<void>(resolve => upstream.listen(0, '127.0.0.1', resolve));
  const address = upstream.address();
  if (!address || typeof address === 'string') throw new Error('local upstream fixture could not bind');
  const createdIDs: string[] = [];
  try {
    const response = await page.request.get('/api/v1/admin/connector-presets');
    expect(response.ok()).toBeTruthy();
    const { metadata } = await response.json();
    expect(metadata).toHaveLength(4);
    await page.goto('/admin/connectors');
    await expect(page.getByRole('button', { name: '이 프리셋으로 연결', exact: true })).toHaveCount(4);
    for (const preset of metadata) {
      const index = metadata.findIndex((p: { id: string }) => p.id === preset.id);
      await page.getByRole('button', { name: '이 프리셋으로 연결', exact: true }).nth(index).click();
      const dialog = page.getByRole('dialog');
      await expect(dialog.getByText('선택하여 저장할 공식 항목', { exact: true })).toBeVisible();
      const required = preset.fields.filter((f: { required: boolean }) => f.required);
      await expect(dialog.locator('.official-field-grid input[type=checkbox]:disabled')).toHaveCount(required.length);
      await dialog.getByRole('button', { name: '필수 항목만 선택', exact: true }).click();
      await expect(dialog.locator('.official-field-grid input[type=checkbox]:checked')).toHaveCount(required.length);
      if (preset.id === 'work24-occupation-summary') {
        const code = dialog.getByLabel('직업 코드');
        await expect(code).toBeEmpty();
        await expect(code).toHaveAttribute('required', '');
        expect(await code.evaluate((element: HTMLInputElement) => element.checkValidity())).toBe(false);
        await code.fill('133301');
      }
      if (preset.id === 'work24-ncs') {
        const narrative = dialog.getByLabel('수행직무내용');
        await expect(narrative).toBeEmpty();
        await expect(narrative).toHaveAttribute('required', '');
        await expect(dialog.getByLabel('조회 능력단위 수', { exact: true })).toHaveValue('5');
        await narrative.fill('관리자가 작성한 합성 일반 직무 설명');
      }
      await expect(dialog.getByLabel('데이터 형식', { exact: true })).toBeDisabled();
      await expect(dialog.getByLabel('데이터 용도', { exact: true })).toBeDisabled();
      await expect(dialog.getByLabel('인증키 쿼리 매개변수', { exact: true })).toHaveValue('authKey');
      const name = `브라우저 공식형식 검증 ${preset.id} ${Date.now()}`;
      await dialog.getByLabel('연결 이름').fill(name);
      await dialog.getByLabel('Endpoint URL').fill(`http://127.0.0.1:${address.port}/${preset.id}`);
      await dialog.getByLabel('API 인증키', { exact: true }).fill(token);
      await dialog.getByRole('switch', { name: '연결 활성화', exact: true }).uncheck();
      await page.setViewportSize({ width: 390, height: 844 });
      await noHorizontalOverflow(page, `mobile preset ${preset.id}`);
      const saved = page.waitForResponse(r => r.url().endsWith('/api/v1/admin/connectors') && r.request().method() === 'POST');
      await dialog.getByRole('button', { name: '연결 저장', exact: true }).click();
      const saveResponse = await saved;
      expect(saveResponse.status()).toBe(200);
      const connector = await saveResponse.json();
      createdIDs.push(connector.id);
      expect(connector.apiKey).toBeFalsy();
      expect(connector.hasApiKey).toBe(true);
      expect(connector.selectedFields.sort()).toEqual(required.map((f: { name: string }) => f.name).sort());
      await expect(dialog).toHaveCount(0);
      await page.setViewportSize({ width: 1440, height: 1050 });
      const card = page.locator('.connector-list .panel').filter({ has: page.getByRole('heading', { name, exact: true }) });
      const tested = page.waitForResponse(r => r.url().endsWith(`/api/v1/admin/connectors/${connector.id}/test`) && r.request().method() === 'POST');
      await card.getByRole('button', { name: '연결 테스트', exact: true }).click();
      const testResponse = await tested;
      expect(testResponse.status()).toBe(200);
      const result = await testResponse.json();
      expect(result.count).toBe(1);
      expect(Object.keys(result.preview[0].raw).sort()).toEqual(required.map((f: { name: string }) => f.name).sort());
      expect(result.preview[0].source.kind).toBe('public_api');
      expect(result.preview[0].source.url).toBe(preset.specUrl);
      expect(result.preview[0].source.fields.sort()).toEqual(Object.keys(result.preview[0].raw).sort());
      expect(result.preview[0].salary).toBeUndefined();
      expect(result.preview[0].description).toBeUndefined();
      expect(result.preview[0].tuitionReference).toBeUndefined();
      if (preset.dataset === 'jobs' || preset.dataset === 'training') expect(result.preview[0].skills).toEqual([]);
      if (preset.dataset === 'training') expect(result.preview[0].id).toBe('E2E-C1:3');
      const preview = page.getByRole('dialog');
      await expect(preview.getByRole('heading', { name: '선택하여 보관할 원천 응답', exact: true })).toBeVisible();
      await preview.getByText('출처와 정규화 결과 확인', { exact: true }).click();
      await expect(preview.locator('details pre')).toContainText('public_api');
      await expect(preview).not.toContainText(token);
      await preview.getByRole('button', { name: '닫기', exact: true }).click();
    }
    expect(called.sort()).toEqual(metadata.map((m: { id: string }) => m.id).sort());
    expect(errors).toEqual([]);
  } finally {
    for (const id of createdIDs) {
      expect((await page.request.delete(`/api/v1/admin/connectors/${id}`)).ok()).toBeTruthy();
    }
    await new Promise<void>((resolve, reject) => upstream.close(error => error ? reject(error) : resolve()));
  }
});

test('원천·가공 데이터 및 매핑 관리 화면 모바일 새로고침', async ({ page }) => {
  await administrator(page);
  await page.setViewportSize({ width: 390, height: 844 });
  const errors: string[] = [];
  page.on('pageerror', e => errors.push(e.message));
  for (const route of ['/admin/connectors', '/admin/data', '/admin/mappings', '/admin/data-policy']) {
    await page.goto(route);
    await expect(page.locator('main h1')).toBeVisible();
    await page.reload();
    await expect(page.locator('main h1')).toBeVisible();
    expect(new URL(page.url()).pathname).toBe(route);
    await noHorizontalOverflow(page, route);
    await expect(page.getByText('페이지를 찾을 수 없습니다')).toHaveCount(0);
  }
  expect(errors).toEqual([]);
});

test('새 사용자 명시 동의·경력 저장·철회 시 개인 데이터 삭제', async ({ page, browser, baseURL }) => {
  await administrator(page);
  const suffix = `${Date.now()}-${randomBytes(4).toString('hex')}`;
  const temporaryEmail = `privacy-e2e-${suffix}@example.test`;
  const temporaryPassword = randomBytes(24).toString('base64url');
  const create = await page.request.post('/api/v1/admin/users', { data: { email: temporaryEmail, password: temporaryPassword, name: '일회성 브라우저 동의 검증', role: 'user' } });
  expect(create.ok()).toBeTruthy();
  const user = await create.json();
  const context = await browser.newContext({ baseURL, viewport: { width: 390, height: 844 } });
  try {
    const login = await context.request.post('/api/v1/auth/login', { data: { email: temporaryEmail, password: temporaryPassword } });
    expect(login.ok()).toBeTruthy();
    const personal = await context.newPage();
    const errors: string[] = [];
    personal.on('pageerror', e => errors.push(e.message));
    await personal.goto('/profile');
    const checkbox = personal.getByRole('checkbox', { name: /개인정보 수집·이용 안내를 읽었으며/ });
    const accept = personal.getByRole('button', { name: '동의하고 경력 서비스 이용', exact: true });
    await expect(checkbox).not.toBeChecked();
    await expect(accept).toBeDisabled();
    await expect(personal.getByRole('button', { name: '예시 경력 채우기', exact: true })).toHaveCount(0);
    await checkbox.check();
    await personal.reload();
    await expect(checkbox).not.toBeChecked();
    await expect(accept).toBeDisabled();
    expect((await (await context.request.get('/api/v1/privacy')).json()).consent).toBeNull();
    await checkbox.check();
    await accept.click();
    await personal.getByRole('button', { name: '예시 경력 채우기', exact: true }).click();
    await personal.getByRole('button', { name: '경력 저장', exact: true }).click();
    await expect(personal.getByText('경력 프로필을 저장했습니다.')).toBeVisible();
    expect((await (await context.request.get('/api/v1/profile')).json()).skills.length).toBeGreaterThan(0);
    await personal.goto('/privacy');
    await expect(personal.getByText('현재 안내에 동의함', { exact: true })).toBeVisible();
    await noHorizontalOverflow(personal, 'personal privacy mobile');
    await personal.getByRole('button', { name: '동의 철회 및 경력 데이터 삭제', exact: true }).click();
    await personal.getByRole('dialog').getByRole('button', { name: '동의 철회하고 데이터 삭제', exact: true }).click();
    await expect(accept).toBeDisabled();
    await expect(checkbox).not.toBeChecked();
    const after = await (await context.request.get('/api/v1/profile')).json();
    expect(after.skills || []).toHaveLength(0);
    expect(after.narrative || '').toBe('');
    expect((await (await context.request.get('/api/v1/privacy')).json()).consent).toBeNull();
    await personal.goto('/profile');
    await expect(accept).toBeDisabled();
    await expect(personal.getByRole('button', { name: '예시 경력 채우기', exact: true })).toHaveCount(0);
    expect(errors).toEqual([]);
  } finally {
    // Only this disposable user's career data is removed. The administrator's
    // profile, consent and existing datasets are never changed by these checks.
    await context.request.delete('/api/v1/privacy/consent');
    await context.close();
    expect((await page.request.put(`/api/v1/admin/users/${user.id}`, { data: { name: user.name, role: 'user', disabled: true } })).ok()).toBeTruthy();
  }
});
