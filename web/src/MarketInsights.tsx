import { Badge, Progress } from "@mantine/core";
import { IconChartBar, IconMapPin, IconTrendingUp } from "@tabler/icons-react";
import { Empty, Loading, number, Panel, SourceTag, useResource } from "./App";
type Market = {
  totalJobs: number;
  regions: { name: string; count: number }[];
  skillFrequency: { name: string; count: number }[];
  trends: { name: string; current: number; previous: number; change: number }[];
  salarySamples: { title: string; salary: string; source: any; url: string }[];
  updatedAt?: string;
  hasBaseline: boolean;
};
export default function MarketInsights({ jobId }: { jobId: string }) {
  const market = useResource<Market>(
    jobId ? `/market?jobId=${encodeURIComponent(jobId)}` : null,
  );
  const data = market.data;
  return (
    <Panel
      className="market-panel"
      title="실제 채용 데이터로 보는 기회"
      action={
        <Badge variant="light" color="teal">
          출처 기반 분석
        </Badge>
      }
    >
      <p>
        연결한 데이터 중 합성 예시와 마감된 공고를 제외하고 분석합니다. 현재
        등록된 데이터 범위의 관측값입니다.
      </p>
      <Loading error={market.error} loading={market.loading} />
      {data && data.totalJobs === 0 ? (
        <Empty
          title="실제 채용 데이터가 아직 없습니다"
          description="관리자가 채용정보 연동을 설정하면 지역별 기회, 요구 역량 변화와 공개 급여 정보를 확인할 수 있어요."
        />
      ) : (
        data && (
          <>
            <div className="market-summary">
              <IconChartBar size={26} />
              <span>현재 분석하는 실제 채용 공고</span>
              <strong>
                {number(data.totalJobs)}
                <small>건</small>
              </strong>
            </div>
            <div className="two-col">
              <div>
                <h3 className="market-title">
                  <IconMapPin size={20} />
                  지역별 채용 기회
                </h3>
                {data.regions?.length ? (
                  data.regions.map((region) => (
                    <div className="market-row" key={region.name}>
                      <div>
                        <span>{region.name || "지역 미지정"}</span>
                        <strong>{number(region.count)}건</strong>
                      </div>
                      <Progress
                        value={
                          (region.count / Math.max(1, data.totalJobs)) * 100
                        }
                        color="teal"
                        size={7}
                      />
                    </div>
                  ))
                ) : (
                  <p className="muted">지역 정보가 포함된 공고가 없습니다.</p>
                )}
              </div>
              <div>
                <h3 className="market-title">
                  <IconTrendingUp size={20} />
                  요구 역량의 변화
                </h3>
                {!data.hasBaseline ? (
                  <p className="muted">
                    이전 관측 데이터가 쌓이면 역량별 요구 빈도의 변화를
                    보여드립니다.
                  </p>
                ) : data.trends?.length ? (
                  data.trends.slice(0, 8).map((trend) => (
                    <div className="market-trend" key={trend.name}>
                      <span>{trend.name}</span>
                      <small>
                        {trend.previous} → {trend.current}건
                      </small>
                      <Badge
                        variant="light"
                        color={
                          trend.change > 0
                            ? "teal"
                            : trend.change < 0
                              ? "orange"
                              : "gray"
                        }
                      >
                        {trend.change > 0 ? "+" : ""}
                        {trend.change}건
                      </Badge>
                    </div>
                  ))
                ) : (
                  <p className="muted">
                    현재 비교 가능한 역량 변화가 없습니다.
                  </p>
                )}
              </div>
            </div>
            {data.salarySamples?.length > 0 && (
              <>
                <h3 className="market-title">공개된 급여 정보 참고</h3>
                <div className="salary-list">
                  {data.salarySamples.map((salary, i) => (
                    <div className="salary-row" key={`${salary.title}-${i}`}>
                      <div>
                        <strong>{salary.title}</strong>
                        <SourceTag source={salary.source} />
                      </div>
                      <span>{salary.salary}</span>
                      {salary.url && (
                        <a href={salary.url} target="_blank" rel="noreferrer">
                          원문 ↗
                        </a>
                      )}
                    </div>
                  ))}
                </div>
                <p className="helper">
                  공고에 공개된 원문 값을 표시합니다. 고용 형태, 근무 시간과
                  보상 기준이 다를 수 있습니다.
                </p>
              </>
            )}
            {data.updatedAt && (
              <p className="helper">
                최근 데이터 관측:{" "}
                {new Date(data.updatedAt).toLocaleString("ko-KR")}
              </p>
            )}
          </>
        )
      )}
    </Panel>
  );
}
