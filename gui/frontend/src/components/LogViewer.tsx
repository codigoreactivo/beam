import { useEffect, useRef } from "react";

interface Props {
  lines: string[];
  maxHeight?: string;
}

function colorLine(line: string): string {
  if (/error|fail/i.test(line)) return "text-[var(--color-error)]";
  if (/warn/i.test(line)) return "text-[var(--color-warn)]";
  if (/ok|success|done|complete/i.test(line)) return "text-[var(--color-success)]";
  if (/^\d{2}:\d{2}:\d{2}/.test(line)) return "text-[var(--color-subtext)]";
  return "text-[var(--color-text)]";
}

export default function LogViewer({ lines, maxHeight = "12rem" }: Props) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines]);

  return (
    <div
      className="rounded border overflow-y-auto font-mono text-xs leading-5 p-2"
      style={{
        maxHeight,
        backgroundColor: "#0d1117",
        borderColor: "var(--color-border)",
      }}
    >
      {lines.length === 0 ? (
        <span className="text-[var(--color-muted)]">No logs yet</span>
      ) : (
        lines.map((line, i) => (
          <div key={i} className={colorLine(line)}>
            {line}
          </div>
        ))
      )}
      <div ref={bottomRef} />
    </div>
  );
}
