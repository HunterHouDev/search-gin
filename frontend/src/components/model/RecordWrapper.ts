import { reactive } from 'vue';

interface RecordData {
  [key: string]: unknown;
}

class RecordWrapper {
  private data: RecordData;
  private keys: string[];
  private maxLength: number;

  constructor() {
    this.data = reactive({});
    this.keys = [];
    this.maxLength = 30;
  }

  add(key: string, value: unknown) {
    if (this.data.hasOwnProperty(key)) {
      this.data[key] = value;
      return;
    }
    if (this.keys.length >= this.maxLength) {
      const toDel = this.keys.shift();
      if (toDel !== undefined) {
        delete this.data[toDel];
      }
    }
    this.data[key] = value;
    this.keys.push(key);
  }

  get(key: string) {
    return this.data[key];
  }

  getAll() {
    return this.data;
  }
}

const recordWrapper = new RecordWrapper();

export default recordWrapper;
