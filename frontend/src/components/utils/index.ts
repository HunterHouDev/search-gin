export const formatMovieType = (title: string) => {
  if (title.indexOf('{{') >= 0) {
    title = title.split('{{')[1];
  }
  if (title.indexOf('}}') >= 0) {
    title = title.split('}}')[0];
  }
  return title;
};

export const formatTags = (title: string) => {
  if (title.indexOf('《') >= 0) {
    title = title.split('《')[1];
  }
  if (title.indexOf('》') >= 0) {
    title = title.split('》')[0];
  }
  return title;
};

export const formatSeries = (code: string | undefined) => {
  if (code) {
    if (code.indexOf('-') == 0) {
      return '';
    }
    return code.split('-')[0]?.substring(0, 5);
  }
};

export const formatNumber = (code: string | undefined) => {
  if (code) {
    if (code.indexOf('-') == 0) {
      return code.substring(1);
    }
    return code?.split('-')[1];
  }
};

export const formatCode = (code: string | undefined) => {
  if (code) {
    if (code.indexOf('-') == 0) {
      return (code = code.substring(1));
    }
    return code.substring(0, 10);
  }
  return '';
};

export const formatTitle = (title: string | undefined, length?: number) => {
  if (!title) {
    return '';
  }
  // 移除括号及括号内内容
  title = title.replace(/\([^)]*\)/g, '');
  title = title.replace(/\（[^）]*\）/g, '');
  if (title.lastIndexOf(']') >= 0) {
    title = title.substring(title.lastIndexOf(']') + 1);
  }
  if (title.indexOf('{{') >= 0) {
    title = title.replace(`{{${formatMovieType(title)}}}`, '');
  }
  if (title.indexOf('《') >= 0) {
    title = title.replace(`《${formatTags(title)}》`, '');
  }
  if (length) {
    title = title.substring(0, length);
  }
  return title;
};

export const MovieTypeOptions = [
  { label: '国产', value: '国产' },
  { label: '骑兵', value: '骑兵' },
  { label: '步兵', value: '步兵' },
  { label: '西洋', value: '斯巴达' },
  { label: '漫动', value: '漫动' },
];

export const MovieTypeSelects = [
  ...MovieTypeOptions,
  { label: '无', value: '无' },
  { label: '全部', value: '' },
];

export const DescEnum = [
  { label: '↑', value: 'asc' },
  { label: '↓', value: 'desc' },
];

export const FieldEnum = [
  { label: '时间', value: 'MTime' },
  { label: '大小', value: 'Size' },
  { label: '名称', value: 'Code' },
];

class EEnum {
  label = '';
  value = '';
}

export const getLabelByValue = (value: string, arr: EEnum[]) => {
  if (arr && arr.length > 0) {
    let label = '';
    arr.forEach((item: EEnum) => {
      if (item.value === value) {
        label = item.label;
      }
    });
    return label;
  } else {
    return value;
  }
};

export const parseTime = (baseNum: number) => {
  const hh = (parseInt(String(baseNum / 3600)) + ':').padStart(3, '0');
  const mm = (parseInt(String((baseNum % 3600) / 60)) + ':').padStart(3, '0');
  const ss = Number(baseNum % 60)
    .toFixed(0)
    .padStart(2, '0');
  return hh + mm + ss;
};

export const parseTimeZH = (baseNum: number) => {
  const hh =
    baseNum / 3600 > 0 ? parseInt(String(baseNum / 3600)) + '小时' : '';
  const mm =
    (baseNum % 3600) % 60 > 0
      ? parseInt(String((baseNum % 3600) / 60)) + '分'
      : '';
  const ss = Number(baseNum % 60).toFixed(0) + '秒';
  return hh + mm + ss;
};

export const defaultVideoOffset = [2, 60];
export const defaultVideoWidth = 800;
