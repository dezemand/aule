import { code } from "@streamdown/code";
import { Streamdown } from "streamdown";
import "streamdown/styles.css";

type MarkdownProps = {
  children: string;
  animated?: boolean;
  isAnimating?: boolean;
  className?: string;
};

export function Markdown({
  children,
  animated = false,
  isAnimating = false,
  className,
}: MarkdownProps) {
  return (
    <Streamdown
      animated={animated}
      isAnimating={isAnimating}
      className={`markdown ${className ?? ""}`.trim()}
      plugins={{ code }}
    >
      {children}
    </Streamdown>
  );
}
