import { memo } from "react";
import ReactMarkdown from "react-markdown";
import type { Components } from "react-markdown";
import remarkGfm from "remark-gfm";

interface MarkdownTextProps {
  children: string;
  className?: string;
}

const markdownPlugins = [remarkGfm];
const markdownComponents: Components = {
  a({ href, children, ...props }) {
    return <a href={href} target="_blank" rel="noreferrer" {...props}>{children}</a>;
  },
};

export const MarkdownText = memo(function MarkdownText({ children, className = "" }: MarkdownTextProps) {
  return (
    <div className={`markdown-body ${className}`.trim()}>
      <ReactMarkdown
        remarkPlugins={markdownPlugins}
        components={markdownComponents}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
});
