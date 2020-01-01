import { removePositionInfo } from '//unist/nodes';
import * as toml from '@iarna/toml';
import { BlockContent } from 'mdast';
import { PostNode } from '../post_parser';
import * as mdast from 'mdast';

export const mdRoot = (children: mdast.Content[]): mdast.Root => {
  return { type: 'root', children };
};

export const mdHeading = (
  depth: 1 | 2 | 3 | 4 | 5 | 6,
  children: mdast.PhrasingContent[]
): mdast.Heading => {
  return { type: 'heading', depth, children };
};

export const mdHeading1 = (child: string): mdast.Heading => {
  return { type: 'heading', depth: 1, children: [mdText(child)] };
};

export const mdCode = (code: string): mdast.Code => {
  return { type: 'code', value: code };
};

export const mdCodeWithLang = (lang: string, code: string): mdast.Code => {
  return { type: 'code', lang, value: code };
};

export const mdText = (value: string): mdast.Text => {
  return { type: 'text', value };
};

export const mdPara = (children: mdast.PhrasingContent[]): mdast.Paragraph => {
  return { type: 'paragraph', children };
};

export const mdParaText = (value: string): mdast.Paragraph => {
  return { type: 'paragraph', children: [mdText(value)] };
};

export const mdOrderedList = (children: BlockContent[]): mdast.List => {
  return {
    type: 'list',
    ordered: true,
    spread: false,
    start: 1,
    children: children.map(c => mdListItem([c])),
  };
};

export const mdListItem = (children: mdast.BlockContent[]): mdast.ListItem => {
  return { type: 'listItem', spread: false, checked: undefined, children };
};

export const mdFrontmatterToml = (value: toml.JsonMap): mdast.Content => {
  let raw = toml
    .stringify(value)
    .trimEnd()
    .replace(/T00:00:00.000Z/, '');
  // The typings for mdast don't allow anything except a whitelist.
  return ({
    type: 'toml',
    value: raw,
  } as unknown) as mdast.Content;
};

export const stripPositions = (node: PostNode): PostNode => {
  removePositionInfo(node.node);
  return node;
};
