/** A coloured run of text: [role, text, bold]. role is a colour token name; '' is `text`. */
export type Segment = [role: string, text: string, bold?: boolean];
export type Line = Segment[];
export type Frame = 'heavy' | 'round' | 'square' | 'double' | 'dashed' | 'rule';
export type Section = [title: string, right: Line | null, lines: Line[]];

export interface MonitorTUI {
  /** 'level-low' under 50, 'level-mid' 50–79, 'level-high' 80 and up. */
  level(percent: number): string;
  Box(title: string, right: Line | null, width: number, height: number, lines: Line[], frame?: Frame): Line[];
  Tile(title: string, value: Line, history: number[] | null, max: number, role: string, width: number): Line[];
  Bar(percent: number, width: number, role?: string, on?: string, off?: string): Line;
  Sparkline(values: number[], width: number, max: number): string;
  BrailleGraph(values: number[], width: number, height: number, max: number, role?: string): Line[];
  Header(m: { name: string; e: number; p: number; gpuCores: number; battery?: string; clock: string; have: boolean }, width: number): Line[];
  ProcessTable(rows: { pid: number; command: string; cpu: number; gpu: number; mem: number; rss: number }[], width: number, count?: number): Line[];
  RunBars(a: number[], b: number[], height: number, max: number, barWidth: number, roleA: string, roleB: string): Line[];
  SectionedFrame(leftWidth: number, rightWidth: number, bands: { left: Section; right: Section }[], bottom: [title: string, lines: Line[]]): Line[];
  StatRow(label: string, value: Line, percent: number | null, innerWidth: number, labelWidth?: number): Line;
  /** A dim `── title ───` subheading, width cells wide. */
  rule(title: string, width: number): Line;
  hjoin(gap: number, ...blocks: Line[][]): Line[];
  center(line: Line, width: number): Line;
  spread(left: Line, right: Line, width: number): Line;
  /** Render lines into a <pre>, adding the mt-screen class. */
  Screen(el: HTMLElement, lines: Line[]): HTMLElement;
}
declare global { interface Window { MonitorTUI: MonitorTUI } }
