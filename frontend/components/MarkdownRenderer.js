import ReactMarkdown from 'react-markdown';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';
import rehypeRaw from 'rehype-raw';

const MarkdownRenderer = ({ markdownContent }) => {
  if (!markdownContent) {
    return null;
  }

  return (
    <ReactMarkdown
      remarkPlugins={[remarkMath]}
      rehypePlugins={[rehypeRaw, rehypeKatex]}
    >
      {markdownContent}
    </ReactMarkdown>
  );
};

export default MarkdownRenderer;