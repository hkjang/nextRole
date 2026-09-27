import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  Alert,
  Badge,
  Button,
  Checkbox,
  Modal,
  Textarea,
  TextInput,
} from "@mantine/core";
import { IconCheck, IconShieldLock, IconTrash } from "@tabler/icons-react";
import { api } from "./api";
import { Loading, PageTitle, Panel, useApp, useResource } from "./App";
export type PrivacyState = {
  notice: string;
  version: string;
  consent: { version: string; acceptedAt: string } | null;
};
export function usePrivacy() {
  const state = useResource<PrivacyState>("/privacy");
  return {
    ...state,
    accepted:
      !!state.data?.consent?.acceptedAt &&
      state.data.consent.version === state.data.version,
  };
}
export function PrivacyNotice({
  privacy,
  allowWithdraw = false,
  onAccepted,
}: {
  privacy: ReturnType<typeof usePrivacy>;
  allowWithdraw?: boolean;
  onAccepted?: () => void;
}) {
  const { notify } = useApp(),
    [checked, setChecked] = useState(false),
    [busy, setBusy] = useState(false),
    [withdraw, setWithdraw] = useState(false);
  useEffect(() => setChecked(false), [privacy.data?.version]);
  async function accept() {
    if (!checked || !privacy.data) return;
    setBusy(true);
    try {
      await api("/privacy/consent", "POST", {
        version: privacy.data.version,
        accepted: true,
      });
      setChecked(false);
      await privacy.reload();
      onAccepted?.();
      notify("개인정보 수집·이용 동의를 기록했습니다.");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy(false);
    }
  }
  async function revoke() {
    setBusy(true);
    try {
      await api("/privacy/consent", "DELETE");
      setWithdraw(false);
      setChecked(false);
      await privacy.reload();
      notify("동의를 철회하고 관련 개인 경력 데이터를 삭제했습니다.");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <Loading loading={privacy.loading} error={privacy.error} />
      {privacy.data && (
        <Panel
          title="개인정보 수집·이용 안내"
          action={
            <Badge variant="light">안내 버전 {privacy.data.version}</Badge>
          }
        >
          <div className="privacy-notice" tabIndex={0}>
            {privacy.data.notice}
          </div>
          <Alert
            color="teal"
            mt="lg"
            icon={<IconShieldLock size={20} />}
            title="경력 분석에 필요한 정보만 입력하세요"
          >
            주민등록번호, 전화번호, 이메일, 상세 주소 등 불필요한 식별정보는
            입력하거나 업로드하지 마세요. 식별정보를 분석 입력에서 제외하도록
            처리하며, 외부 AI 사용 여부와 이용 범위는 위 안내를 확인해 주세요.
          </Alert>
          {privacy.accepted ? (
            <>
              <div className="privacy-status">
                <Badge color="teal">현재 안내에 동의함</Badge>
                <span>
                  동의일{" "}
                  {new Date(privacy.data.consent!.acceptedAt).toLocaleString(
                    "ko-KR",
                  )}
                </span>
              </div>
              {allowWithdraw && (
                <Button
                  variant="light"
                  color="red"
                  leftSection={<IconTrash size={18} />}
                  onClick={() => setWithdraw(true)}
                >
                  동의 철회 및 경력 데이터 삭제
                </Button>
              )}
            </>
          ) : (
            <div className="consent-actions">
              {privacy.data.consent && (
                <Alert color="orange">
                  수집·이용 안내가 변경되었습니다. 현재 버전을 확인하고 다시
                  동의해 주세요.
                </Alert>
              )}
              <Checkbox
                label="개인정보 수집·이용 안내를 읽었으며, 해당 버전의 수집·이용에 동의합니다."
                checked={checked}
                onChange={(e) => setChecked(e.target.checked)}
                size="md"
              />
              <p>
                동의하지 않아도 안내를 확인할 수 있습니다. 경력 입력·분석·저장을
                이용하려면 동의가 필요합니다.
              </p>
              <Button
                disabled={!checked}
                loading={busy}
                onClick={accept}
                leftSection={<IconCheck size={18} />}
              >
                동의하고 경력 서비스 이용
              </Button>
            </div>
          )}
        </Panel>
      )}
      <Modal
        opened={withdraw}
        onClose={() => setWithdraw(false)}
        title="동의 철회 및 데이터 삭제"
        centered
      >
        <p>
          동의를 철회하면 저장된 경력 프로필, 시뮬레이션과 관련 개인 경력
          데이터가 삭제됩니다. 삭제된 데이터는 복구할 수 없습니다.
        </p>
        <p>
          계정의 로그인 정보와 서비스 운영에 필요한 기록의 보관 범위는 수집·이용
          안내를 확인해 주세요.
        </p>
        <div className="button-row">
          <Button variant="default" onClick={() => setWithdraw(false)}>
            취소
          </Button>
          <Button color="red" loading={busy} onClick={revoke}>
            동의 철회하고 데이터 삭제
          </Button>
        </div>
      </Modal>
    </>
  );
}
export function PrivacyPage() {
  const privacy = usePrivacy();
  return (
    <>
      <PageTitle
        title="내 경력 정보와 동의 관리"
        description="어떤 정보를 어떤 목적으로 사용하는지 확인하고, 수집·이용 동의를 관리하세요."
      />
      <PrivacyNotice privacy={privacy} allowWithdraw />
      {privacy.accepted && (
        <Button component={Link} to="/profile" variant="light">
          내 경력으로 이동
        </Button>
      )}
    </>
  );
}
export function DataPolicyPage() {
  const resource = useResource<{ notice: string; version: string }>(
      "/admin/data-policy",
    ),
    { notify } = useApp(),
    [notice, setNotice] = useState(""),
    [version, setVersion] = useState(""),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    if (resource.data) {
      setNotice(resource.data.notice);
      setVersion(String(resource.data.version));
    }
  }, [resource.data]);
  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api("/admin/data-policy", "PUT", { notice, version });
      await resource.reload();
      notify("개인정보 수집·이용 안내를 저장했습니다.");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <PageTitle
        eyebrow="DATA GOVERNANCE"
        title="개인정보 수집·이용 안내"
        description="수집 목적, 항목, 보유 기간과 이용자 권리를 서비스 운영 방식에 맞게 안내하세요."
      />
      <Loading loading={resource.loading} error={resource.error} />
      {resource.data && (
        <div className="settings-layout">
          <Panel title="이용자에게 표시할 안내">
            <form className="form-stack" onSubmit={save}>
              <TextInput
                required
                label="안내 버전"
                description="내용이 달라지면 새 버전을 입력하세요. 이용자는 새 버전에 다시 동의해야 합니다."
                value={version}
                onChange={(e) => setVersion(e.target.value)}
              />
              <Textarea
                required
                label="개인정보 수집·이용 안내 전문"
                minRows={18}
                value={notice}
                onChange={(e) => setNotice(e.target.value)}
                description="수집 목적·항목, 보유 및 삭제 기준, 동의 거부 시 제한, 철회 방법, AI 제공자·외부 전송 여부, 담당 연락처를 포함하세요."
              />
              <Button type="submit" loading={busy}>
                안내 저장
              </Button>
            </form>
          </Panel>
          <Panel title="동의 처리 기준" className="settings-help">
            <p>
              동의 체크박스는 미리 선택되지 않습니다. 사용자가 안내를 읽고 직접
              선택하면 안내 버전과 동의 시각을 기록합니다.
            </p>
            <p>
              현재 버전에 동의하기 전에는 경력 입력·파일 분석·저장을 진행하지
              않습니다.
            </p>
            <p>
              이용자는 개인정보·동의 메뉴에서 동의를 철회하고 관련 경력 데이터를
              삭제할 수 있습니다.
            </p>
          </Panel>
        </div>
      )}
    </>
  );
}

export function CareerConsentGate({ children }: { children: React.ReactNode }) {
  const privacy = usePrivacy();
  return privacy.accepted ? (
    <>{children}</>
  ) : (
    <>
      <PageTitle
        title="나의 경력 정보, 동의 후 안전하게"
        description="경력 분석과 시뮬레이션을 시작하기 전에 수집·이용 안내를 확인해 주세요."
      />
      <PrivacyNotice privacy={privacy} />
    </>
  );
}
