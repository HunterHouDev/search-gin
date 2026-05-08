class RecordWrapper<T extends Record<string, unknown>> {
  private record: T;
  private array: Array<keyof T>;
  private maxLength: 30;

  constructor() {
    this.record = {} as T;
    this.array = [];
    this.maxLength = 30;
  }
  add(key: string, value: unknown) {
    if (this.record.hasOwnProperty(key)) {
      (this.record as Record<string, unknown>)[key] = value;
      return;
    }
    if (this.array.length >= this.maxLength) {
      const toDel = this.array.shift();
      // 确保toDel不为undefined，避免类型错误
      if (toDel !== undefined) {
        this.record.hasOwnProperty(toDel) && delete this.record[toDel];
      }
    }
    // 由于类型“T”是泛型的，只能编制索引以供读取，这里使用类型断言来绕过类型检查
    (this.record as Record<string, unknown>)[key] = value;
    this.array.push(key);
  }
  get(key: string) {
    return this.record[key];
  }
}
const recordWrapper = new RecordWrapper();

export default recordWrapper;
