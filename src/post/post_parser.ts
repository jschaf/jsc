// Parser for post content.
import remarkParse from 'remark-parse';
import unified from 'unified';
import * as unist from 'unist';
import nodeRemove from 'unist-util-remove';
import { checkState } from '//asserts';
import { Unzipper } from '//zip_files';
import { PostMetadata } from '//post/post_metadata';

export const TEXT_PACK_BUNDLE_PREFIX = 'Content.textbundle';

export class PostParser {
  private readonly processor: unified.Processor<unified.Settings>;

  private constructor() {
    this.processor = unified().use(remarkParse, { commonmark: true });
  }

  static create(): PostParser {
    return new PostParser();
  }

  async parseTextPack(textPack: Buffer): Promise<PostNode> {
    const entries = await Unzipper.unzip(textPack);
    const texts = entries.filter(
      e => e.filePath === TEXT_PACK_BUNDLE_PREFIX + '/text.md'
    );
    checkState(
      texts.length === 1,
      'Expected a single text.md file in TextPack.'
    );
    const text = texts[0];
    return this.parseMarkdown(text.contents.toString('utf8'));
  }

  parseMarkdown(markdown: string): PostNode {
    const node = this.processor.parse(markdown);
    const metadata = PostMetadata.parseFromMarkdownAST(node);
    const nodeSansMetadata = nodeRemove(node, PostMetadata.isMetadataNode) || {
      type: 'root',
    };
    return new PostNode(metadata, nodeSansMetadata);
  }
}

export class PostNode {
  constructor(readonly metadata: PostMetadata, readonly node: unist.Node) {}
}