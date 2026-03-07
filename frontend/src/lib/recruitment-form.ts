import type { CreateRecruitmentSubmissionRequest } from "@/lib/types";

export type RecruitmentFormValue = CreateRecruitmentSubmissionRequest;

const REQUIRED_FIELDS: Array<{ key: keyof RecruitmentFormValue; label: string }> = [
  { key: "name", label: "姓名" },
  { key: "school", label: "学校" },
  { key: "grade", label: "年级" },
  { key: "direction", label: "方向" },
  { key: "contact", label: "联系方式" },
  { key: "bio", label: "个人简介" },
];

export function getMissingRecruitmentFields(value: RecruitmentFormValue): string[] {
  return REQUIRED_FIELDS.filter(({ key }) => !value[key].trim()).map(({ label }) => label);
}

export function normalizeRecruitmentFormValue(value: RecruitmentFormValue): RecruitmentFormValue {
  return {
    name: value.name.trim(),
    school: value.school.trim(),
    grade: value.grade.trim(),
    direction: value.direction.trim(),
    contact: value.contact.trim(),
    bio: value.bio.trim(),
  };
}
