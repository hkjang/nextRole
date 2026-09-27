'use strict';
const toggle = document.querySelector('.menu-toggle');
const navigation = document.querySelector('#nav');
toggle?.addEventListener('click', () => {
  const open = toggle.getAttribute('aria-expanded') !== 'true';
  toggle.setAttribute('aria-expanded', String(open));
  toggle.setAttribute('aria-label', open ? '탐색 메뉴 닫기' : '탐색 메뉴 열기');
  navigation.classList.toggle('open', open);
});
navigation?.addEventListener('click', (event) => {
  if (event.target.closest('a')) {navigation.classList.remove('open');toggle?.setAttribute('aria-expanded', 'false');toggle?.setAttribute('aria-label', '탐색 메뉴 열기');}
});
document.addEventListener('keydown', (event) => {
  if (event.key === 'Escape' && toggle?.getAttribute('aria-expanded') === 'true') {navigation.classList.remove('open');toggle.setAttribute('aria-expanded', 'false');toggle.setAttribute('aria-label', '탐색 메뉴 열기');toggle.focus();}
});
const screens = {
  simulator: ['NextRole 경력 시뮬레이터 실제 화면: 추가 역량에 따른 적합도 변화와 학습 경로', '목표 직무와 추가 역량을 바꾸며 결과를 비교합니다. 화면의 데이터는 합성 시연 예시입니다.'],
  compare: ['NextRole 직무 비교 실제 화면: 목표 직무 세 개의 적합도와 전환 요건', '최대 세 개의 목표 직무를 같은 경력으로 비교합니다. 화면의 데이터는 합성 시연 예시입니다.'],
  roadmap: ['NextRole 나의 로드맵 실제 화면: 기간별 실행계획과 완료 체크', '3·6·12개월 계획으로 나누고, 완료한 실행 항목을 기록합니다.'],
  dashboard: ['NextRole 대시보드 실제 화면: 내 경력 현황과 추천 직무', '내 경력에서 출발해 다음 직무의 가능성을 탐색합니다. 화면의 데이터는 합성 시연 예시입니다.']
};
const tabs = Array.from(document.querySelectorAll('[data-screen]'));
function selectTab(tab) {
  const name = tab.dataset.screen;
  if (!screens[name]) return;
  tabs.forEach(button => {button.setAttribute('aria-selected', String(button === tab));button.tabIndex = button === tab ? 0 : -1;});
  const screenshot = document.querySelector('#product-screen');
  screenshot.src = `screenshots/${name}.png`;
  screenshot.alt = screens[name][0];
  document.querySelector('#screen-panel').setAttribute('aria-labelledby', tab.id);
  document.querySelector('#screen-caption').textContent = screens[name][1];
}
tabs.forEach(tab => {
  tab.addEventListener('click', () => selectTab(tab));
  tab.addEventListener('keydown', event => {
    const index = tabs.indexOf(tab);
    let next;
    if(event.key === 'ArrowRight') next = tabs[(index + 1) % tabs.length];
    if(event.key === 'ArrowLeft') next = tabs[(index - 1 + tabs.length) % tabs.length];
    if(event.key === 'Home') next = tabs[0];
    if(event.key === 'End') next = tabs[tabs.length - 1];
    if(next){event.preventDefault();selectTab(next);next.focus();}
  });
});
